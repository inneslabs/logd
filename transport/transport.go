/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package transport

import (
	"context"
	"fmt"
	"net"
	"net/netip"
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
	rateLimitEvery = time.Microsecond * 100
	rateLimitBurst = 10
)

type Sub struct {
	raddrPort   netip.AddrPort
	lastPing    time.Time
	queryParams *cmd.QueryParams
}

type Transporter struct {
	Out         chan *ProtoPair
	laddrPort   string
	conn        *net.UDPConn
	subs        map[string]*Sub
	subsMu      sync.RWMutex
	rateLimiter *rate.Limiter
	readSecret  []byte
	writeSecret []byte
	buf         *ring.RingBuffer
	alarmSvc    *alarm.Svc
}
type Config struct {
	LaddrPort   string
	ReadSecret  string
	WriteSecret string
	Buf         *ring.RingBuffer
	AlarmSvc    *alarm.Svc
}

type Packet struct {
	Data      []byte
	RaddrPort netip.AddrPort
}

type ProtoPair struct {
	Msg   *cmd.Msg
	Bytes []byte
}

func NewTransporter(cfg *Config) *Transporter {
	t := &Transporter{
		Out:         make(chan *ProtoPair, 1),
		laddrPort:   cfg.LaddrPort,
		subs:        make(map[string]*Sub),
		subsMu:      sync.RWMutex{},
		rateLimiter: rate.NewLimiter(rate.Every(rateLimitEvery), rateLimitBurst),
		readSecret:  []byte(cfg.ReadSecret),
		writeSecret: []byte(cfg.WriteSecret),
		buf:         cfg.Buf,
		alarmSvc:    cfg.AlarmSvc,
	}

	return t
}

func (t *Transporter) SetReadSecret(secret []byte) {
	t.readSecret = secret
}

func (t *Transporter) SetWriteSecret(secret []byte) {
	t.writeSecret = secret
}

func (t *Transporter) Listen(ctx context.Context) {
	l, err := net.ResolveUDPAddr("udp", t.laddrPort)
	if err != nil {
		panic(fmt.Errorf("resolve laddr err: %w", err))
	}
	t.conn, err = net.ListenUDP("udp", l)
	if err != nil {
		panic(fmt.Errorf("listen udp err: %w", err))
	}
	defer t.conn.Close()
	fmt.Println("listening udp on", t.conn.LocalAddr())

	// one gopher reads packets
	packets := make(chan *Packet)
	go func() {
		fmt.Println("packet-reading gopher started")
		for {
			t.readPacket(packets)
		}
	}()

	// some gophers handle packets
	for i := 0; i < 10; i++ {
		go func(i int) {
			fmt.Printf("packet-handling gopher %d started\n", i+1)
			for {
				t.handlePacket(<-packets)
			}
		}(i)
	}

	// one gopher writes to the subs
	go t.writeToSubs()

	// one gopher kicks subs that don't ping
	go t.kickLateSubs()

	// wait for the party to end
	<-ctx.Done()
	fmt.Println("stopped listening udp")
}

func (t *Transporter) readPacket(packets chan<- *Packet) {
	buf := make([]byte, 2048)
	t.conn.SetReadDeadline(time.Now().Add(time.Second))
	n, raddrPort, err := t.conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		return
	}
	packets <- &Packet{
		Data:      buf[:n],
		RaddrPort: raddrPort,
	}
}

func (t *Transporter) handlePacket(packet *Packet) {
	unpk, err := auth.UnpackSignedData(packet.Data)
	if err != nil {
		fmt.Println("unpack msg err:", err)
		return
	}
	c := &cmd.Cmd{}
	err = proto.Unmarshal(unpk.Payload, c)
	if err != nil {
		fmt.Println("protobuf unmarshal err:", err)
		return
	}
	switch c.GetName() {
	case cmd.Name_WRITE:
		err = t.handleWrite(c, unpk)
	case cmd.Name_TAIL:
		err = t.handleTail(c, packet.RaddrPort, unpk)
	case cmd.Name_PING:
		err = t.handlePing(packet.RaddrPort, unpk)
	case cmd.Name_QUERY:
		err = t.handleQuery(c, packet.RaddrPort, unpk)
	}
	if err != nil {
		fmt.Println("handle packet err:", err)
	}
}

func (t *Transporter) writeToSubs() {
	for {
		protoPair := <-t.Out
		t.subsMu.RLock()
		for raddr, sub := range t.subs {
			if !shouldSendToSub(sub, protoPair) {
				continue
			}
			t.rateLimiter.Wait(context.Background())
			_, err := t.conn.WriteToUDPAddrPort(protoPair.Bytes, sub.raddrPort)
			if err != nil {
				fmt.Printf("write udp err: (%s) %s\n", raddr, err)
			}
		}
		t.subsMu.RUnlock()
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
