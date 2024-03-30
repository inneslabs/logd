package udp

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/inneslabs/fnpool"
	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/guard"
	"github.com/inneslabs/logd/sign"
	"github.com/inneslabs/logd/store"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

type Cfg struct {
	WorkerPoolSize      int           `yaml:"worker_pool_size"`
	LaddrPort           string        `yaml:"laddr_port"`
	Guard               *guard.Cfg    `yaml:"guard"`
	ReadSecret          string        `yaml:"read_secret"`
	WriteSecret         string        `yaml:"write_secret"`
	SubRateLimitEvery   time.Duration `yaml:"sub_rate_limit_every"`
	SubRateLimitBurst   int           `yaml:"sub_rate_limit_burst"`
	QueryRateLimitEvery time.Duration `yaml:"query_rate_limit_every"`
	QueryRateLimitBurst int           `yaml:"query_rate_limit_burst"`
	LogStore            *store.Store
	Ctx                 context.Context
}

const (
	MaxPacketSize         = 1920
	ReplyKey              = "//logd"
	PingPeriod            = time.Second * 2
	KickAfterMissingPings = 3
)

type UdpSvc struct {
	ctx              context.Context
	laddrPort        string
	conn             *net.UDPConn
	subs             map[string]*Sub
	subsMu           sync.RWMutex
	forSubs          chan *ProtoPair
	subRateLimiter   *rate.Limiter
	queryRateLimiter *rate.Limiter
	readSecret       []byte
	writeSecret      []byte
	logStore         *store.Store
	pkgPool          *sync.Pool
	workerPool       *fnpool.Pool
	guard            *guard.Guard
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

func NewSvc(cfg *Cfg) *UdpSvc {
	svc := &UdpSvc{
		ctx:       cfg.Ctx,
		laddrPort: cfg.LaddrPort,
		guard:     guard.NewGuard(cfg.Guard),
		subs:      make(map[string]*Sub),
		subsMu:    sync.RWMutex{},
		// increased buffer size from 4 (2024-02-11)
		forSubs:          make(chan *ProtoPair, 100),
		subRateLimiter:   rate.NewLimiter(rate.Every(cfg.SubRateLimitEvery), cfg.SubRateLimitBurst),
		queryRateLimiter: rate.NewLimiter(rate.Every(cfg.QueryRateLimitEvery), cfg.QueryRateLimitBurst),
		readSecret:       []byte(cfg.ReadSecret),
		writeSecret:      []byte(cfg.WriteSecret),
		logStore:         cfg.LogStore,
		pkgPool: &sync.Pool{
			New: func() any {
				return &sign.Pkg{
					Sum:       make([]byte, 32), // sha256
					TimeBytes: make([]byte, 8),  // uint64
					Payload:   make([]byte, MaxPacketSize),
				}
			},
		},
		workerPool: fnpool.NewPool(cfg.WorkerPoolSize),
	}
	go svc.listen()
	go svc.kickLateSubs()
	go svc.writeToSubs()
	return svc
}

func (svc *UdpSvc) listen() {
	l, err := net.ResolveUDPAddr("udp", svc.laddrPort)
	if err != nil {
		panic(fmt.Errorf("resolve laddr err: %w", err))
	}
	svc.conn, err = net.ListenUDP("udp", l)
	if err != nil {
		panic(fmt.Errorf("listen udp err: %w", err))
	}
	defer svc.conn.Close()
	go func() {
		for {
			svc.readPacket()
		}
	}()
	fmt.Println("listening udp on", svc.conn.LocalAddr())
	<-svc.ctx.Done()
	fmt.Println("packet handler pool stopping...")
	svc.workerPool.StopAndWait()
	fmt.Println("packet handler pool stopped")
	fmt.Println("udp svc shutdown")
}

func (svc *UdpSvc) readPacket() {
	buf := make([]byte, MaxPacketSize)
	n, raddr, err := svc.conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		return
	}
	svc.handlePacket(&Packet{
		Data:  buf[:n],
		Raddr: raddr,
	})
}

func (svc *UdpSvc) handlePacket(packet *Packet) {
	svc.workerPool.Dispatch(func() {
		pkg, _ := svc.pkgPool.Get().(*sign.Pkg)
		err := sign.UnpackSignedData(packet.Data, pkg)
		if err != nil {
			return
		}
		c := &cmd.Cmd{}
		err = proto.Unmarshal(pkg.Payload, c)
		if err != nil {
			return
		}
		switch c.GetName() {
		case cmd.Name_WRITE:
			svc.handleWrite(c, pkg)
		case cmd.Name_TAIL:
			svc.handleTail(c, packet.Raddr, pkg)
		case cmd.Name_PING:
			svc.handlePing(packet.Raddr, pkg)
		case cmd.Name_QUERY:
			svc.handleQuery(c, packet.Raddr, pkg)
		}
		svc.pkgPool.Put(pkg)
	})
}

func (svc *UdpSvc) writeToSubs() {
	for {
		select {
		case protoPair := <-svc.forSubs:
			svc.subsMu.RLock()
			for raddr, sub := range svc.subs {
				if !shouldSendToSub(sub, protoPair.Msg) {
					continue
				}
				svc.subRateLimiter.Wait(svc.ctx)
				_, err := svc.conn.WriteToUDPAddrPort(protoPair.Bytes, sub.raddr)
				if err != nil {
					fmt.Printf("write udp err: (%s) %s\n", raddr, err)
				}
			}
			svc.subsMu.RUnlock()
		case <-svc.ctx.Done():
			fmt.Println("writeToSubs ended")
			return
		}
	}
}

func shouldSendToSub(sub *Sub, msg *cmd.Msg) bool {
	if sub.queryParams != nil {
		keyPrefix := sub.queryParams.GetKeyPrefix()
		if keyPrefix != "" && !strings.HasPrefix(msg.GetKey(), keyPrefix) {
			return false
		}
		qLvl := sub.queryParams.GetLvl()
		if qLvl != cmd.Lvl_LVL_UNKNOWN && qLvl > msg.GetLvl() {
			return false
		}
		qResponseStatus := sub.queryParams.GetResponseStatus()
		if qResponseStatus != 0 && qResponseStatus != msg.GetResponseStatus() {
			return false
		}
		qUrl := sub.queryParams.GetUrl()
		if qUrl != "" && !strings.HasPrefix(msg.GetUrl(), qUrl) {
			return false
		}
	}
	return true
}

func (svc *UdpSvc) reply(txt string, raddr netip.AddrPort) {
	payload, _ := proto.Marshal(&cmd.Msg{
		Key: ReplyKey,
		Txt: &txt,
	})
	svc.subRateLimiter.Wait(svc.ctx)
	svc.conn.WriteToUDPAddrPort(payload, raddr)
}

func (svc *UdpSvc) kickLateSubs() {
	for {
		select {
		case <-time.After(PingPeriod):
			for _, sub := range svc.subs {
				if sub.lastPing.Before(time.Now().Add(-(PingPeriod * KickAfterMissingPings))) {
					svc.subsMu.Lock()
					delete(svc.subs, sub.raddr.String())
					svc.subsMu.Unlock()
					fmt.Printf("kicked %s\n", sub.raddr.String())
					svc.reply("kick", sub.raddr)
				}
			}
		case <-svc.ctx.Done():
			fmt.Println("kickLateSubs ended")
			return
		}
	}
}
