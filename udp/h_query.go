package udp

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/sign"
	"google.golang.org/protobuf/proto"
)

const (
	hardLimit = 100000
	EndMsg    = "+END"
)

func (svc *UdpSvc) handleQuery(command *cmd.Cmd, raddr netip.AddrPort, pkg *sign.Pkg) {
	valid, err := svc.signer.Verify(svc.readSecret, pkg)
	if !valid || err != nil {
		return
	}
	if !svc.guard.Good(pkg.Sum) {
		return
	}
	svc.queryRateLimiter.Wait(svc.ctx)
	query := command.GetQueryParams()
	offset := query.GetOffset()
	limit := limit(query.GetLimit())
	keyPrefix := query.GetKeyPrefix()
	for log := range svc.logStore.Read(keyPrefix, offset, limit) {
		msg := &cmd.Msg{}
		err = proto.Unmarshal(log, msg)
		if err != nil {
			fmt.Println("query unmarshal protobuf err:", err)
			return
		}
		if matchMsg(msg, query) {
			err := svc.subRateLimiter.Wait(svc.ctx)
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
	txt := query.GetTxt()
	httpMethod := query.GetHttpMethod()
	url := query.GetUrl()
	responseStatus := query.GetResponseStatus()
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
	msgTxt := msg.GetTxt()
	if txt != "" && !strings.Contains(strings.ToLower(msgTxt), strings.ToLower(txt)) {
		return false
	}
	msgHttpMethod := msg.GetHttpMethod()
	if httpMethod != cmd.HttpMethod_METHOD_UNKNOWN && httpMethod != msgHttpMethod {
		return false
	}
	msgUrl := msg.GetUrl()
	if url != "" && !strings.HasPrefix(msgUrl, url) {
		return false
	}
	msgResponseStatus := msg.GetResponseStatus()
	if responseStatus != 0 && responseStatus != msgResponseStatus {
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
