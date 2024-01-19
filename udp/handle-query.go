package udp

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

// handleQuery reads from the head newest first
func (svc *UdpSvc) handleQuery(c *cmd.Cmd, raddrPort netip.AddrPort, unpk *auth.Unpacked) error {
	valid, err := auth.Verify(svc.readSecret, unpk)
	if !valid || err != nil {
		return errors.New("unauthorized")
	}
	svc.queryRateLimiter.Wait(context.TODO())
	limit := limit(c.GetQueryParams(), svc.buf.Size())
	tStart := tStart(c.GetQueryParams())
	tEnd := tEnd(c.GetQueryParams())
	env := c.GetQueryParams().GetEnv()
	cmdSvc := c.GetQueryParams().GetSvc()
	fn := c.GetQueryParams().GetFn()
	lvl := c.GetQueryParams().GetLvl()
	txt := c.GetQueryParams().GetTxt()
	httpMethod := c.GetQueryParams().GetHttpMethod()
	url := c.GetQueryParams().GetUrl()
	responseStatus := c.GetQueryParams().GetResponseStatus()
	max := svc.buf.Size()
	var offset, found uint32
	head := svc.buf.Head()
	for offset < max && found < limit {
		offset++
		payload := svc.buf.ReadOne((head - offset) % max)
		if payload == nil {
			break // reached end of items in non-full buffer
		}
		msg := &cmd.Msg{}
		err = proto.Unmarshal(payload, msg)
		if err != nil {
			fmt.Println("query unmarshal protobuf err:", err)
			continue
		}
		msgT := msg.T.AsTime()
		if tStart != nil && msgT.Before(*tStart) {
			continue
		}
		if tEnd != nil && msgT.After(*tEnd) {
			continue
		}
		if env != "" && env != msg.GetEnv() {
			continue
		}
		if cmdSvc != "" && cmdSvc != msg.GetSvc() {
			continue
		}
		if fn != "" && fn != msg.GetFn() {
			continue
		}
		if lvl != cmd.Lvl_LVL_UNKNOWN && lvl != msg.GetLvl() {
			continue
		}
		msgTxt := msg.GetTxt()
		if txt != "" && !strings.Contains(strings.ToLower(msgTxt), strings.ToLower(txt)) {
			continue
		}
		msgHttpMethod := msg.GetHttpMethod()
		if httpMethod != cmd.HttpMethod_METHOD_UNKNOWN && httpMethod != msgHttpMethod {
			continue
		}
		msgUrl := msg.GetUrl()
		if url != "" && !strings.HasPrefix(msgUrl, url) {
			continue
		}
		msgResponseStatus := msg.GetResponseStatus()
		if responseStatus != 0 && responseStatus != msgResponseStatus {
			continue
		}
		err := svc.connRateLimiter.Wait(context.TODO())
		if err != nil {
			return err
		}
		_, err = svc.conn.WriteToUDPAddrPort(payload, raddrPort)
		if err != nil {
			return err
		}
		found++
	}
	////////////////////////////////////////////////////////////////
	// TODO: find better way to signal end /////////////////////////
	////////////////////////////////////////////////////////////////
	time.Sleep(time.Millisecond * 50) // ensure +END arrives last
	end := "+END"
	endPayload, _ := proto.Marshal(&cmd.Msg{
		Fn:  "logd",
		Txt: &end,
	})
	_, err = svc.conn.WriteToUDPAddrPort(endPayload, raddrPort)
	if err != nil {
		fmt.Println("write to udp err:", err)
	}
	return nil
}

func limit(q *cmd.QueryParams, bufSize uint32) uint32 {
	if q == nil {
		return bufSize
	}
	var limit uint32 = bufSize
	qLimit := q.GetLimit()
	if qLimit != 0 && qLimit < bufSize {
		limit = qLimit
	}
	return limit
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
