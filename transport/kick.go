/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package transport

import (
	"fmt"
	"net"
	"time"

	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

const (
	PingPeriod            = time.Second
	KickAfterMissingPings = 10
)

func (t *Transporter) kickLateSubs(conn *net.UDPConn) {
	for {
		for raddr, sub := range t.subs {
			if sub.lastPing.Before(time.Now().Add(-(PingPeriod * KickAfterMissingPings))) {
				t.kickSub(conn, sub, raddr)
				return
			}
		}
		time.Sleep(PingPeriod)
	}
}

// kickSub removes sub from map
func (t *Transporter) kickSub(conn *net.UDPConn, sub *Sub, raddr string) {
	t.mu.Lock()
	delete(t.subs, raddr)
	t.mu.Unlock()
	fmt.Printf("kicked %s, no ping received, timed out\r\n", raddr)
	txt := "you've been kicked, ping timed out"
	payload, err := proto.Marshal(&cmd.Msg{
		Fn:  "logd",
		Txt: &txt,
	})
	if err != nil {
		fmt.Println("pack msg err:", err)
		return
	}
	_, err = conn.WriteToUDP(payload, sub.raddr)
	if err != nil {
		fmt.Printf("write udp err: (%s) %s\r\n", raddr, err)
	}
}
