package transport

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func (t *Transporter) handleQuery(c *cmd.Cmd, conn *net.UDPConn, raddr *net.UDPAddr, sum, timeBytes, payload []byte) error {
	valid, err := auth.Verify(t.readSecret, sum, timeBytes, payload)
	if !valid || err != nil {
		return fmt.Errorf("%s unauthorised to query: %w", raddr.IP.String(), err)
	}
	limit := limit(c.GetQueryParams(), t.buf.Size())
	tStart := tStart(c.GetQueryParams())
	tEnd := tEnd(c.GetQueryParams())
	env := c.GetQueryParams().GetEnv()
	svc := c.GetQueryParams().GetSvc()
	fn := c.GetQueryParams().GetFn()
	lvl := c.GetQueryParams().GetLvl()
	txt := c.GetQueryParams().GetTxt()
	httpMethod := c.GetQueryParams().GetHttpMethod()
	url := c.GetQueryParams().GetUrl()
	responseStatus := c.GetQueryParams().GetResponseStatus()
	max := t.buf.Size()
	var offset, found uint32
	for offset < max && found < limit {
		payload := t.buf.ReadOne(offset)
		offset++
		msg := &cmd.Msg{}
		err = proto.Unmarshal(*payload, msg)
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
		if svc != "" && svc != msg.GetSvc() {
			continue
		}
		if fn != "" && fn != msg.GetFn() {
			continue
		}
		if lvl != cmd.Lvl_LVL_UNKNOWN && lvl != msg.GetLvl() {
			continue
		}
		msgTxt := msg.GetTxt()
		if txt != "" && msgTxt != "" && !strings.Contains(msgTxt, txt) {
			continue
		}
		msgHttpMethod := msg.GetHttpMethod()
		if httpMethod != cmd.HttpMethod_METHOD_UNKNOWN && msgHttpMethod != cmd.HttpMethod_METHOD_UNKNOWN && httpMethod != msgHttpMethod {
			continue
		}
		msgUrl := msg.GetUrl()
		if url != "" && msgUrl != "" && !strings.Contains(msgUrl, url) {
			continue
		}
		msgResponseStatus := msg.GetResponseStatus()
		if responseStatus != 0 && msgResponseStatus != 0 && responseStatus != msgResponseStatus {
			continue
		}
		_, err = conn.WriteToUDP(*payload, raddr)
		if err != nil {
			fmt.Printf("write udp err: (%s) %s\r\n", raddr, err)
		}
		found++
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
