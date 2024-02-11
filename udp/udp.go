/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package udp

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

const MaxPacketSize = 1920

type UdpSvc struct {
	laddrPort        string
	conn             *net.UDPConn
	subs             map[string]*Sub
	subsMu           sync.RWMutex
	forSubs          chan *ProtoPair
	subRateLimiter   *rate.Limiter
	queryRateLimiter *rate.Limiter
	readSecret       []byte
	writeSecret      []byte
	ringBuf          *ring.RingBuffer
	unpkPool         *sync.Pool
	alarmSvc         *alarm.Svc
}

type Config struct {
	LaddrPort           string
	ReadSecret          string
	WriteSecret         string
	RingBuf             *ring.RingBuffer
	AlarmSvc            *alarm.Svc
	SubRateLimitEvery   time.Duration
	SubRateLimitBurst   int
	QueryRateLimitEvery time.Duration
	QueryRateLimitBurst int
}

type Sub struct {
	raddr       netip.AddrPort
	lastPing    time.Time
	queryParams *cmd.QueryParams
}

type Packet struct {
	Data  []byte
	Raddr netip.AddrPort
}

type ProtoPair struct {
	Msg   *cmd.Msg
	Bytes []byte
}

func NewSvc(cfg *Config) *UdpSvc {
	return &UdpSvc{
		laddrPort:        cfg.LaddrPort,
		subs:             make(map[string]*Sub),
		subsMu:           sync.RWMutex{},
		forSubs:          make(chan *ProtoPair, 4), // small buffer helps a lot
		subRateLimiter:   rate.NewLimiter(rate.Every(cfg.SubRateLimitEvery), cfg.SubRateLimitBurst),
		queryRateLimiter: rate.NewLimiter(rate.Every(cfg.QueryRateLimitEvery), cfg.QueryRateLimitBurst),
		readSecret:       []byte(cfg.ReadSecret),
		writeSecret:      []byte(cfg.WriteSecret),
		ringBuf:          cfg.RingBuf,
		alarmSvc:         cfg.AlarmSvc,
		unpkPool: &sync.Pool{
			New: func() any {
				return &auth.Unpacked{
					Sum:       make([]byte, auth.SumLen),
					TimeBytes: make([]byte, auth.TimeLen),
					Payload:   make([]byte, MaxPacketSize),
				}
			},
		},
	}
}

func (svc *UdpSvc) Listen(ctx context.Context) {
	l, err := net.ResolveUDPAddr("udp", svc.laddrPort)
	if err != nil {
		panic(fmt.Errorf("resolve laddr err: %w", err))
	}
	svc.conn, err = net.ListenUDP("udp", l)
	if err != nil {
		panic(fmt.Errorf("listen udp err: %w", err))
	}
	defer svc.conn.Close()
	fmt.Println("listening udp on", svc.conn.LocalAddr())

	// one gopher reads packets
	packets := make(chan *Packet, 4)
	go func() {
		fmt.Printf("packet-reading gopher started\n")
		for {
			svc.readPacket(packets)
		}
	}()

	// one gopher handles packets
	go func() {
		fmt.Printf("packet-handling gopher started\n")
		for {
			svc.handlePacket(<-packets)
		}
	}()

	// one gopher writes to the subs
	go svc.writeToSubs()

	// one gopher kicks subs that don't ping
	go svc.kickLateSubs()

	// wait for the gopher party to end
	<-ctx.Done()
	fmt.Println("stopped listening udp")
}

func (svc *UdpSvc) readPacket(packets chan<- *Packet) {
	buf := make([]byte, MaxPacketSize)
	n, raddr, err := svc.conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		return
	}
	packets <- &Packet{
		Data:  buf[:n],
		Raddr: raddr,
	}
}

func (svc *UdpSvc) handlePacket(packet *Packet) {
	// get a *Unpacked from pool
	unpk, _ := svc.unpkPool.Get().(*auth.Unpacked)
	err := auth.UnpackSignedData(packet.Data, unpk)
	if err != nil {
		return
	}
	c := &cmd.Cmd{}
	err = proto.Unmarshal(unpk.Payload, c)
	if err != nil {
		return
	}
	// ignore errors
	switch c.GetName() {
	case cmd.Name_WRITE:
		svc.handleWrite(c, unpk)
	case cmd.Name_TAIL:
		svc.handleTail(c, packet.Raddr, unpk)
	case cmd.Name_PING:
		svc.handlePing(packet.Raddr, unpk)
	case cmd.Name_QUERY:
		svc.handleQuery(c, packet.Raddr, unpk)
	}
	// return *Unpacked to pool
	svc.unpkPool.Put(unpk)
}

func (svc *UdpSvc) writeToSubs() {
	for {
		protoPair := <-svc.forSubs
		svc.subsMu.RLock()
		for raddr, sub := range svc.subs {
			if !shouldSendToSub(sub, protoPair) {
				continue
			}
			svc.subRateLimiter.Wait(context.Background())
			_, err := svc.conn.WriteToUDPAddrPort(protoPair.Bytes, sub.raddr)
			if err != nil {
				fmt.Printf("write udp err: (%s) %s\n", raddr, err)
			}
		}
		svc.subsMu.RUnlock()
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

func (svc *UdpSvc) reply(txt string, raddr netip.AddrPort) {
	payload, _ := proto.Marshal(&cmd.Msg{
		Fn:  "logd",
		Txt: &txt,
	})
	svc.subRateLimiter.Wait(context.Background())
	svc.conn.WriteToUDPAddrPort(payload, raddr)
}
