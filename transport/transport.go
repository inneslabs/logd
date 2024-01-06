package transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

const bufferSize = 2048

type Sub struct {
	raddr    *net.UDPAddr
	lastPing time.Time
}

type Transporter struct {
	In          chan *[]byte
	Out         chan *[]byte
	subs        map[string]*Sub
	mu          sync.Mutex
	readSecret  []byte
	writeSecret []byte
}

func NewTransporter() *Transporter {
	return &Transporter{
		In:   make(chan *[]byte, 50),
		Out:  make(chan *[]byte, 50),
		subs: make(map[string]*Sub),
		mu:   sync.Mutex{},
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
