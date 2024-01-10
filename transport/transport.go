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
	"github.com/swissinfo-ch/logd/ring"
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
