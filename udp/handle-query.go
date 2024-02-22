package udp

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

const hardLimit = 1000

func (udpSvc *UdpSvc) handleQuery(query *cmd.Cmd, raddr netip.AddrPort, unpk *auth.Unpacked) error {
	valid, err := auth.Verify(udpSvc.readSecret, unpk)
	if !valid || err != nil {
		return errors.New("unauthorized")
	}

	udpSvc.queryRateLimiter.Wait(udpSvc.ctx)

	offset := query.GetQueryParams().GetOffset()
	limit := limit(query.GetQueryParams().GetLimit())
	keyPrefix := query.GetQueryParams().GetKeyPrefix()

	if keyPrefix == "" {
		keyPrefix = "/"
	}

	for log := range udpSvc.logStore.Read(keyPrefix, offset, limit) {
		msg := &cmd.Msg{}
		err = proto.Unmarshal(log, msg)
		if err != nil {
			fmt.Println("query unmarshal protobuf err:", err)
			return err
		}
		if matchMsg(msg, query) {
			err := udpSvc.subRateLimiter.Wait(udpSvc.ctx)
			if err != nil {
				return err
			}
			_, err = udpSvc.conn.WriteToUDPAddrPort(log, raddr)
			if err != nil {
				return err
			}
		}
	}

	time.Sleep(time.Millisecond * 50) // ensure +END arrives last
	udpSvc.reply("+END", raddr)
	return nil
}

func matchMsg(msg *cmd.Msg, query *cmd.Cmd) bool {
	tStart := tStart(query.GetQueryParams())
	tEnd := tEnd(query.GetQueryParams())
	lvl := query.GetQueryParams().GetLvl()
	txt := query.GetQueryParams().GetTxt()
	httpMethod := query.GetQueryParams().GetHttpMethod()
	url := query.GetQueryParams().GetUrl()
	responseStatus := query.GetQueryParams().GetResponseStatus()
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
