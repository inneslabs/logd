package transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/swissinfo-ch/logd/msg"
)

const bufferSize = 2048

type Transporter struct {
	In          chan []byte
	Out         chan []byte
	subs        map[string]*net.UDPAddr
	mu          sync.Mutex
	readSecret  []byte
	writeSecret []byte
}

func NewTransporter() *Transporter {
	return &Transporter{
		In:   make(chan []byte),
		Out:  make(chan []byte, 10),
		subs: make(map[string]*net.UDPAddr),
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
			buf = make([]byte, bufferSize)
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, raddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			sum, payload, err := unpackMsg(buf[:n])
			if err != nil {
				fmt.Println("unpack msg err:", err)
				continue
			}
			// if tailing, first msg is "tail"
			if string(payload) == "tail" {
				go t.handleTailer(raddr, sum, payload)
				continue
			}
			t.In <- payload
		}
	}
}

func (t *Transporter) handleTailer(raddr *net.UDPAddr, sum, payload []byte) {
	valid, err := validateSum(t.readSecret, sum, payload)
	if err != nil || !valid {
		return
	}
	t.mu.Lock()
	t.subs[raddr.AddrPort().String()] = raddr
	t.mu.Unlock()
	time.Sleep(time.Millisecond * 50)
	e := &msg.Msg{
		Fn:  "logd",
		Msg: fmt.Sprintf("tailer %s joined", raddr),
	}
	data, err := cbor.Marshal(e)
	if err != nil {
		fmt.Println("handle tailer: cbor marshal err:", err)
		return
	}
	t.Out <- data
}

func (t *Transporter) writeToConn(ctx context.Context, conn *net.UDPConn) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-t.Out:
			for raddr, sub := range t.subs {
				_, err := conn.WriteToUDP(msg, sub)
				if err != nil {
					fmt.Printf("write udp err: (%s) %s\r\n", raddr, err)
					t.mu.Lock()
					delete(t.subs, raddr)
					t.mu.Unlock()
				}
			}
		}
	}
}
