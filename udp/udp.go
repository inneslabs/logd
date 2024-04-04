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

const (
	MaxPacketSize     = 1024
	ReplyKey          = "//logd"
	EndMsg            = "+END"
	PingPeriod        = 2 * time.Second
	PingLossTolerance = 3
	QueryHardLimit    = 100000
)

type Cfg struct {
	LaddrPort string     `yaml:"laddr_port"`
	Guard     *guard.Cfg `yaml:"guard"`
	Secrets   *Secrets   `yaml:"secrets"`
	LogStore  *store.Store
}

type Secrets struct {
	Read  string `yaml:"read"`
	Write string `yaml:"write"`
}

type UdpSvc struct {
	laddrPort string
	conn      *net.UDPConn
	tails     map[string]*tail
	ping      chan string
	newTail   chan *tail
	write     chan *cmd.Msg
	secrets   *Secrets
	logStore  *store.Store
	pkgPool   *sync.Pool
	guard     *guard.Guard
}

type tail struct {
	raddr       netip.AddrPort
	lastPing    time.Time
	queryParams *cmd.QueryParams
}

func NewSvc(ctx context.Context, cfg *Cfg) *UdpSvc {
	svc := &UdpSvc{
		laddrPort: cfg.LaddrPort,
		guard:     guard.NewGuard(ctx, cfg.Guard),
		tails:     make(map[string]*tail),
		write:     make(chan *cmd.Msg, 100),
		ping:      make(chan string, 1),
		newTail:   make(chan *tail, 1),
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
	go svc.theThing()
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
	go func() {
		defer svc.conn.Close()
		for {
			err := svc.readPacket()
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	fmt.Println("listening udp on", svc.conn.LocalAddr())
}

func (svc *UdpSvc) readPacket() error {
	buf := make([]byte, MaxPacketSize)
	n, raddr, err := svc.conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		return err
	}
	// get a pointer to a reusable pkg.Pkg to unpack packet
	p, _ := svc.pkgPool.Get().(*pkg.Pkg)
	defer svc.pkgPool.Put(p)
	err = pkg.Unpack(buf[:n], p)
	if err != nil {
		return err
	}
	c := &cmd.Cmd{}
	err = proto.Unmarshal(p.Payload, c)
	if err != nil {
		return err
	}
	switch c.Name {
	case cmd.Name_WRITE:
		if !svc.guard.Good([]byte(svc.secrets.Write), p) {
			return nil
		}
		svc.write <- c.Msg
	case cmd.Name_TAIL:
		if !svc.guard.Good([]byte(svc.secrets.Read), p) {
			return nil
		}
		svc.newTail <- &tail{
			raddr:       raddr,
			lastPing:    time.Now(),
			queryParams: c.GetQueryParams(),
		}
		svc.reply("\rtailing logs\033[0K", raddr)
	case cmd.Name_PING:
		if !svc.guard.Good([]byte(svc.secrets.Read), p) {
			return nil
		}
		svc.ping <- raddr.String()
	case cmd.Name_QUERY:
		if !svc.guard.Good([]byte(svc.secrets.Read), p) {
			return nil
		}
		go svc.handleQuery(c, raddr)
	}
	return nil
}

func (svc *UdpSvc) theThing() {
	for {
		select {
		case msg := <-svc.write:
			svc.handleWrite(msg)
		case ping := <-svc.ping:
			tail, ok := svc.tails[ping]
			if ok {
				tail.lastPing = time.Now()
			}
		case newTail := <-svc.newTail:
			svc.tails[newTail.raddr.String()] = newTail
		case <-time.After(PingPeriod):
			for _, tail := range svc.tails {
				threshold := time.Now().Add(-(PingPeriod * PingLossTolerance))
				if tail.lastPing.Before(threshold) {
					delete(svc.tails, tail.raddr.String())
					fmt.Printf("kicked %s\n", tail.raddr.String())
					svc.reply("kick", tail.raddr)
				}
			}
		}
	}
}

func (svc *UdpSvc) handleWrite(msg *cmd.Msg) error {
	segments := strings.Split(msg.Key, "/")
	if len(segments) < 3 {
		return fmt.Errorf("invalid key %q: too few segments", msg.Key)
	}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("err marshaling proto msg: %w", err)
	}
	// IMPORTANT:
	// This is currently how msg keys are mapped to the rings
	storeKey := fmt.Sprintf("/%s/%s", segments[1], segments[2])
	svc.logStore.Write(storeKey, msgBytes)
	for raddr, tail := range svc.tails {
		if !shouldSendToTail(tail, msg) {
			continue
		}
		_, err := svc.conn.WriteToUDPAddrPort(msgBytes, tail.raddr)
		if err != nil {
			return fmt.Errorf("err writing to %s: %v", raddr, err)
		}
	}
	return nil
}

func shouldSendToTail(t *tail, msg *cmd.Msg) bool {
	if t.queryParams != nil {
		keyPrefix := t.queryParams.GetKeyPrefix()
		if keyPrefix != "" && !strings.HasPrefix(msg.GetKey(), keyPrefix) {
			return false
		}
		qLvl := t.queryParams.GetLvl()
		if qLvl != cmd.Lvl_LVL_UNKNOWN && qLvl > msg.GetLvl() {
			return false
		}
	}
	return true
}

func (svc *UdpSvc) reply(txt string, raddr netip.AddrPort) {
	fmt.Printf("reply to %s: %q\n", raddr, txt)
	payload, err := proto.Marshal(&cmd.Msg{
		Key: ReplyKey,
		Txt: txt,
	})
	if err != nil {
		fmt.Printf("err marshaling proto msg: %v\n", err)
		return
	}
	_, err = svc.conn.WriteToUDPAddrPort(payload, raddr)
	if err != nil {
		fmt.Printf("err replying to %s: %v\n", raddr, err)
		return
	}
}

func (svc *UdpSvc) handleQuery(command *cmd.Cmd, raddr netip.AddrPort) {
	query := command.GetQueryParams()
	keyPrefix := query.GetKeyPrefix()
	offset := query.GetOffset()
	limit := query.GetLimit()
	if limit > QueryHardLimit {
		limit = QueryHardLimit
	}
	for log := range svc.logStore.Read(keyPrefix, offset, limit) {
		msg := &cmd.Msg{}
		err := proto.Unmarshal(log, msg)
		if err != nil {
			fmt.Println("query unmarshal protobuf err:", err)
			return
		}
		if msgMatchesQuery(msg, query) {
			// possibly wait here a few microseconds
			// before sending to prevent packet loss
			_, err = svc.conn.WriteToUDPAddrPort(log, raddr)
			if err != nil {
				return
			}
		}
	}
	time.Sleep(15 * time.Millisecond) // ensure +END arrives last
	svc.reply(EndMsg, raddr)
}

func msgMatchesQuery(msg *cmd.Msg, query *cmd.QueryParams) bool {
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
