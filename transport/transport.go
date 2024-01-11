/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package transport

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"github.com/swissinfo-ch/logd/ring"
	"google.golang.org/protobuf/proto"
)

const (
	socketBufferSize      = 2048
	socketBufferThreshold = 0.75
	rateLimit             = time.Microsecond * 50
)

type Sub struct {
	raddr       *net.UDPAddr
	lastPing    time.Time
	queryParams *cmd.QueryParams
}

type Transporter struct {
	Out         chan *ProtoPair
	subs        map[string]*Sub
	mu          sync.Mutex
	readSecret  []byte
	writeSecret []byte
	buf         *ring.RingBuffer
	alarmSvc    *alarm.Svc
}
type TransporterConfig struct {
	ReadSecret  string
	WriteSecret string
	Buf         *ring.RingBuffer
	AlarmSvc    *alarm.Svc
}

type ProtoPair struct {
	Msg   *cmd.Msg
	Bytes *[]byte
}

func NewTransporter(cfg *TransporterConfig) *Transporter {
	return &Transporter{
		Out:         make(chan *ProtoPair, 1),
		subs:        make(map[string]*Sub),
		mu:          sync.Mutex{},
		readSecret:  []byte(cfg.ReadSecret),
		writeSecret: []byte(cfg.WriteSecret),
		buf:         cfg.Buf,
		alarmSvc:    cfg.AlarmSvc,
	}
}

func (t *Transporter) SetReadSecret(secret []byte) {
	t.readSecret = secret
}

func (t *Transporter) SetWriteSecret(secret []byte) {
	t.writeSecret = secret
}

func (t *Transporter) Listen(ctx context.Context, laddr string) {
	l, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		panic(fmt.Errorf("resolve laddr err: %w", err))
	}
	conn, err := net.ListenUDP("udp", l)
	if err != nil {
		panic(fmt.Errorf("listen udp err: %w", err))
	}
	err = conn.SetReadBuffer(socketBufferSize)
	if err != nil {
		panic(fmt.Errorf("set read buffer size err: %w", err))
	}
	err = conn.SetWriteBuffer(socketBufferSize)
	if err != nil {
		panic(fmt.Errorf("set write buffer size err: %w", err))
	}
	defer conn.Close()
	fmt.Println("listening udp on", conn.LocalAddr())
	go t.readFromConn(ctx, conn)
	go t.writeToSubs(ctx, conn)
	go t.kickLateSubs(conn)
	<-ctx.Done()
	fmt.Println("stopped listening udp")
}

func (t *Transporter) readFromConn(ctx context.Context, conn *net.UDPConn) {
	var buf []byte
	for {
		select {
		case <-ctx.Done():
			return
		default:
			buf = make([]byte, socketBufferSize)
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, raddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			if n >= socketBufferSize*socketBufferThreshold {
				fmt.Printf("warning: socket buffer is >= %f full\r\n", socketBufferThreshold)
			}
			go t.handlePacket(buf[:n], conn, raddr)
		}
	}
}

func (t *Transporter) handlePacket(data []byte, conn *net.UDPConn, raddr *net.UDPAddr) {
	sum, timeBytes, payload, err := auth.UnpackSignedData(data)
	if err != nil {
		fmt.Println("unpack msg err:", err)
		return
	}
	c := &cmd.Cmd{}
	err = proto.Unmarshal(payload, c)
	if err != nil {
		fmt.Println("protobuf unmarshal err:", err)
		return
	}
	switch c.GetName() {
	case cmd.Name_WRITE:
		t.handleWrite(c, raddr, sum, timeBytes, payload)
	case cmd.Name_TAIL:
		t.handleTail(c, conn, raddr, sum, timeBytes, payload)
	case cmd.Name_PING:
		t.handlePing(raddr, sum, timeBytes, payload)
	case cmd.Name_QUERY:
		t.handleQuery(c, conn, raddr, sum, timeBytes, payload)
	}
}

func (t *Transporter) writeToSubs(ctx context.Context, conn *net.UDPConn) {
	for {
		select {
		case <-ctx.Done():
			return
		case protoPair := <-t.Out:
			for raddr, sub := range t.subs {
				if sub.queryParams != nil {
					qEnv := sub.queryParams.GetEnv()
					if qEnv != "" && qEnv != protoPair.Msg.GetEnv() {
						continue
					}
					qSvc := sub.queryParams.GetSvc()
					if qSvc != "" && qSvc != protoPair.Msg.GetSvc() {
						continue
					}
					qFn := sub.queryParams.GetFn()
					if qFn != "" && qFn != protoPair.Msg.GetFn() {
						continue
					}
					qLvl := sub.queryParams.GetLvl()
					if qLvl != cmd.Lvl_LVL_UNKNOWN && qLvl > protoPair.Msg.GetLvl() {
						continue
					}
					qResponseStatus := sub.queryParams.GetResponseStatus()
					if qResponseStatus != 0 && qResponseStatus != protoPair.Msg.GetResponseStatus() {
						continue
					}
					qUrl := sub.queryParams.GetUrl()
					if qUrl != "" && !strings.HasPrefix(protoPair.Msg.GetUrl(), qUrl) {
						continue
					}
				}
				_, err := conn.WriteToUDP(*protoPair.Bytes, sub.raddr)
				if err != nil {
					fmt.Printf("write udp err: (%s) %s\r\n", raddr, err)
				}
				time.Sleep(rateLimit)
			}
		}
	}
}
