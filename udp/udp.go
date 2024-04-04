package udp

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/guard"
	"github.com/inneslabs/logd/pkg"
	"github.com/inneslabs/logd/store"
	"google.golang.org/protobuf/proto"
)

type Cfg struct {
	WorkerPoolSize int        `yaml:"worker_pool_size"`
	LaddrPort      string     `yaml:"laddr_port"`
	Guard          *guard.Cfg `yaml:"guard"`
	Secrets        *Secrets   `yaml:"secrets"`
	LogStore       *store.Store
	Ctx            context.Context
}

type Secrets struct {
	Read  string `yaml:"read"`
	Write string `yaml:"write"`
}

const (
	MaxPacketSize         = 1024
	ReplyKey              = "//logd"
	PingPeriod            = time.Second * 2
	KickAfterMissingPings = 3
)

type UdpSvc struct {
	ctx       context.Context
	laddrPort string
	conn      *net.UDPConn
	tails     map[string]*Tail
	ping      chan string
	newTail   chan *Tail
	forSubs   chan *ProtoPair
	secrets   *Secrets
	logStore  *store.Store
	pkgPool   *sync.Pool
	guard     *guard.Guard
}

type Tail struct {
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
		tails:     make(map[string]*Tail),
		forSubs:   make(chan *ProtoPair, 1),
		ping:      make(chan string, 1),
		newTail:   make(chan *Tail, 1),
		secrets:   cfg.Secrets,
		logStore:  cfg.LogStore,
		pkgPool: &sync.Pool{
			New: func() any {
				return &pkg.Pkg{
					Sum:       make([]byte, 32), // sha256
					TimeBytes: make([]byte, 8),  // uint64
					Payload:   make([]byte, MaxPacketSize),
				}
			},
		},
	}
	go svc.listen()
	go svc.tailReadWrite()
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
	fmt.Println("udp svc shutdown")
}

func (svc *UdpSvc) readPacket() {
	buf := make([]byte, MaxPacketSize)
	n, raddr, err := svc.conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		return
	}
	p, _ := svc.pkgPool.Get().(*pkg.Pkg)
	err = pkg.Unpack(buf[:n], p)
	if err != nil {
		return
	}
	c := &cmd.Cmd{}
	err = proto.Unmarshal(p.Payload, c)
	if err != nil {
		return
	}
	switch c.GetName() {
	case cmd.Name_WRITE:
		if !svc.guard.Good([]byte(svc.secrets.Write), p) {
			break
		}
		svc.handleWrite(c)
	case cmd.Name_TAIL:
		if !svc.guard.Good([]byte(svc.secrets.Read), p) {
			break
		}
		svc.newTail <- &Tail{
			raddr:       raddr,
			lastPing:    time.Now(),
			queryParams: c.GetQueryParams(),
		}
		svc.reply("\rtailing logs\033[0K", raddr)
		fmt.Println("got new tail", raddr.String())
	case cmd.Name_PING:
		if !svc.guard.Good([]byte(svc.secrets.Read), p) {
			break
		}
		svc.ping <- raddr.String()
	case cmd.Name_QUERY:
		if svc.guard.Good([]byte(svc.secrets.Read), p) {
			svc.handleQuery(c, raddr)
		}
	}
	svc.pkgPool.Put(p)
}

func (svc *UdpSvc) tailReadWrite() {
	for {
		select {
		case protoPair := <-svc.forSubs:
			for raddr, tail := range svc.tails {
				if !shouldSendToTail(tail, protoPair.Msg) {
					continue
				}
				_, err := svc.conn.WriteToUDPAddrPort(protoPair.Bytes, tail.raddr)
				if err != nil {
					fmt.Printf("write udp err: (%s) %s\n", raddr, err)
				}
			}
		case ping := <-svc.ping:
			tail, ok := svc.tails[ping]
			if ok {
				tail.lastPing = time.Now()
			}
		case newTail := <-svc.newTail:
			svc.tails[newTail.raddr.String()] = newTail
		case <-time.After(PingPeriod):
			for _, tail := range svc.tails {
				if tail.lastPing.Before(time.Now().Add(-(PingPeriod * KickAfterMissingPings))) {
					delete(svc.tails, tail.raddr.String())
					fmt.Printf("kicked %s\n", tail.raddr.String())
					svc.reply("kick", tail.raddr)
				}
			}
		case <-svc.ctx.Done():
			fmt.Println("writeToSubs ended")
			return
		}
	}
}

func shouldSendToTail(tail *Tail, msg *cmd.Msg) bool {
	if tail.queryParams != nil {
		keyPrefix := tail.queryParams.GetKeyPrefix()
		if keyPrefix != "" && !strings.HasPrefix(msg.GetKey(), keyPrefix) {
			return false
		}
		qLvl := tail.queryParams.GetLvl()
		if qLvl != cmd.Lvl_LVL_UNKNOWN && qLvl > msg.GetLvl() {
			return false
		}
	}
	return true
}

func (svc *UdpSvc) reply(txt string, raddr netip.AddrPort) {
	payload, _ := proto.Marshal(&cmd.Msg{
		Key: ReplyKey,
		Txt: txt,
	})
	svc.conn.WriteToUDPAddrPort(payload, raddr)
}

func (svc *UdpSvc) handleWrite(c *cmd.Cmd) {
	msgBytes, err := proto.Marshal(c.Msg)
	if err != nil {
		return
	}
	segments := strings.Split(c.Msg.GetKey(), "/")
	if len(segments) < 3 {
		return
	}
	// IMPORTANT:
	// This is currently how msg keys are mapped to the rings
	storeKey := fmt.Sprintf("/%s/%s", segments[1], segments[2])
	svc.logStore.Write(storeKey, msgBytes)
	svc.forSubs <- &ProtoPair{
		Msg:   c.Msg,
		Bytes: msgBytes,
	}
}

const (
	hardLimit = 100000
	EndMsg    = "+END"
)

func (svc *UdpSvc) handleQuery(command *cmd.Cmd, raddr netip.AddrPort) {
	query := command.GetQueryParams()
	offset := query.GetOffset()
	limit := limit(query.GetLimit())
	keyPrefix := query.GetKeyPrefix()
	for log := range svc.logStore.Read(keyPrefix, offset, limit) {
		msg := &cmd.Msg{}
		err := proto.Unmarshal(log, msg)
		if err != nil {
			fmt.Println("query unmarshal protobuf err:", err)
			return
		}
		if matchMsg(msg, query) {
			// possibly wait here a few microseconds
			// before sending to prevent packet loss
			_, err = svc.conn.WriteToUDPAddrPort(log, raddr)
			if err != nil {
				return
			}
		}
	}
	time.Sleep(10 * time.Millisecond) // ensure +END arrives last
	svc.reply(EndMsg, raddr)
}

func matchMsg(msg *cmd.Msg, query *cmd.QueryParams) bool {
	keyPrefix := query.GetKeyPrefix()
	if keyPrefix != "" && !strings.HasPrefix(msg.GetKey(), keyPrefix) {
		return false
	}
	tStart := tStart(query)
	tEnd := tEnd(query)
	lvl := query.GetLvl()
	msgT := msg.T.AsTime()
	if tStart != nil && msgT.Before(*tStart) {
		return false
	}
	if tEnd != nil && msgT.After(*tEnd) {
		return false
	}
	if lvl != cmd.Lvl_LVL_UNKNOWN && lvl != msg.GetLvl() {
		return false
	}
	return true
}

func limit(qLimit uint32) uint32 {
	if qLimit != 0 && qLimit < hardLimit {
		return qLimit
	}
	return hardLimit
}

func tStart(q *cmd.QueryParams) *time.Time {
	if q == nil {
		return nil
	}
	tStartPtr := q.GetTStart()
	if tStartPtr == nil {
		return nil
	}
	tStart := tStartPtr.AsTime()
	return &tStart
}

func tEnd(q *cmd.QueryParams) *time.Time {
	if q == nil {
		return nil
	}
	tEndPtr := q.GetTEnd()
	if tEndPtr == nil {
		return nil
	}
	tEnd := tEndPtr.AsTime()
	return &tEnd
}
