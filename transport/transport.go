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
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

const (
	socketBufferSize      = 2048
	socketBufferThreshold = 0.75
	rateLimitEvery        = time.Microsecond * 50
	rateLimitBurst        = 10
)

type Sub struct {
	raddr       *net.UDPAddr
	lastPing    time.Time
	queryParams *cmd.QueryParams
}

type Transporter struct {
	Out         chan *ProtoPair
	bufferPool  *sync.Pool
	subs        map[string]*Sub
	subsMu      sync.RWMutex
	rateLimiter *rate.Limiter
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
	Bytes []byte
}

func NewTransporter(cfg *TransporterConfig) *Transporter {
	return &Transporter{
		Out: make(chan *ProtoPair, 1),
		bufferPool: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, socketBufferSize)
				return &b
			},
		},
		subs:        make(map[string]*Sub),
		subsMu:      sync.RWMutex{},
		rateLimiter: rate.NewLimiter(rate.Every(rateLimitEvery), rateLimitBurst),
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
	for {
		select {
		case <-ctx.Done():
			return
		default:
			bufPtr := t.bufferPool.Get().(*[]byte)
			buf := *bufPtr
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, raddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				t.bufferPool.Put(bufPtr)
				continue
			}
			if n >= socketBufferSize*socketBufferThreshold {
				fmt.Printf("warning: socket buffer is >= %f full\r\n", socketBufferThreshold)
			}
			go t.handlePacket(buf[:n], conn, raddr, func() {
				t.bufferPool.Put(bufPtr) // Return the buffer to the pool
			})
		}
	}
}

func (t *Transporter) handlePacket(data []byte, conn *net.UDPConn, raddr *net.UDPAddr, done func()) {
	defer done()
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
			t.subsMu.RLock()
			for raddr, sub := range t.subs {
				if !shouldSendToSub(sub, protoPair) {
					continue
				}
				err := t.rateLimiter.Wait(ctx)
				if err != nil {
					fmt.Println("failed to wait for subs limiter:", err)
					continue
				}
				_, err = conn.WriteToUDP(protoPair.Bytes, sub.raddr)
				if err != nil {
					fmt.Printf("write udp err: (%s) %s\r\n", raddr, err)
				}
			}
			t.subsMu.RUnlock()
		}
	}
}

func shouldSendToSub(sub *Sub, protoPair *ProtoPair) bool {
	if sub.queryParams != nil {
		qEnv := sub.queryParams.GetEnv()
		if qEnv != "" && qEnv != protoPair.Msg.GetEnv() {
			return false
		}
		qSvc := sub.queryParams.GetSvc()
		if qSvc != "" && qSvc != protoPair.Msg.GetSvc() {
			return false
		}
		qFn := sub.queryParams.GetFn()
		if qFn != "" && qFn != protoPair.Msg.GetFn() {
			return false
		}
		qLvl := sub.queryParams.GetLvl()
		if qLvl != cmd.Lvl_LVL_UNKNOWN && qLvl > protoPair.Msg.GetLvl() {
			return false
		}
		qResponseStatus := sub.queryParams.GetResponseStatus()
		if qResponseStatus != 0 && qResponseStatus != protoPair.Msg.GetResponseStatus() {
			return false
		}
		qUrl := sub.queryParams.GetUrl()
		if qUrl != "" && !strings.HasPrefix(protoPair.Msg.GetUrl(), qUrl) {
			return false
		}
	}
	return true
}
