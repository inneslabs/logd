/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"github.com/swissinfo-ch/logd/ring"
	"google.golang.org/protobuf/proto"
)

const (
	ChanBufferSize   = 1 // payloads to buffer throughout logd
	socketBufferSize = 2048
)

type Sub struct {
	raddr    *net.UDPAddr
	lastPing time.Time
}

type Transporter struct {
	Out         chan *[]byte
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

func NewTransporter(cfg *TransporterConfig) *Transporter {
	return &Transporter{
		Out:         make(chan *[]byte, ChanBufferSize),
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
	defer conn.Close()
	fmt.Println("listening udp on", conn.LocalAddr())
	go t.readFromConn(ctx, conn)
	go t.writeToConn(ctx, conn)
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
			sum, timeBytes, payload, err := auth.UnpackSignedMsg(buf[:n])
			if err != nil {
				fmt.Println("unpack msg err:", err)
				continue
			}
			c := &cmd.Cmd{}
			err = proto.Unmarshal(payload, c)
			if err != nil {
				fmt.Println("protobuf unmarshal err:", err)
				continue
			}
			switch c.GetName() {
			case cmd.Name_WRITE:
				go t.handleWrite(c, raddr, sum, timeBytes, payload)
			case cmd.Name_TAIL:
				go t.handleTail(conn, raddr, sum, timeBytes, payload)
			case cmd.Name_PING:
				go t.handlePing(raddr, sum, timeBytes, payload)
			case cmd.Name_QUERY:
				go t.handleQuery(c, conn, raddr, sum, timeBytes, payload)
			}
		}
	}
}

func (t *Transporter) writeToConn(ctx context.Context, conn *net.UDPConn) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-t.Out:
			for raddr, sub := range t.subs {
				go func(msg *[]byte, sub *Sub, raddr string) {
					_, err := conn.WriteToUDP(*msg, sub.raddr)
					if err != nil {
						fmt.Printf("write udp err: (%s) %s\r\n", raddr, err)
					}
				}(msg, sub, raddr)
			}
		}
	}
}
