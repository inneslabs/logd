package udp

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/inneslabs/logd/cmd"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

const (
	hardLimit = 100000
	EndMsg    = "+END"
)

func (svc *UdpSvc) handleQuery(command *cmd.Cmd, raddr netip.AddrPort) {
	query := command.GetQueryParams()
	offset := query.GetOffset()
	limit := limit(query.GetLimit())
	keyPrefix := query.GetKeyPrefix()
	rateLimit := rate.NewLimiter(svc.queryRateLimit, svc.queryRateLimitBurst)
	for log := range svc.logStore.Read(keyPrefix, offset, limit) {
		msg := &cmd.Msg{}
		err := proto.Unmarshal(log, msg)
		if err != nil {
			fmt.Println("query unmarshal protobuf err:", err)
			return
		}
		if matchMsg(msg, query) {
			err := rateLimit.Wait(svc.ctx)
			if err != nil {
				return
			}
			_, err = svc.conn.WriteToUDPAddrPort(log, raddr)
			if err != nil {
				return
			}
		}
	}
	time.Sleep(time.Millisecond * 10) // ensure +END arrives last
	rateLimit.Wait(svc.ctx)
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
