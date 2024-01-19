/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package udp

import (
	"fmt"
	"net"
	"time"
)

const (
	PingPeriod            = time.Second
	kickAfterMissingPings = 5
)

func (svc *UdpSvc) kickLateSubs() {
	for {
		for raddr, sub := range svc.subs {
			if sub.lastPing.Before(time.Now().Add(-(PingPeriod * kickAfterMissingPings))) {
				svc.kickSub(svc.conn, sub, raddr)
				return
			}
		}
		time.Sleep(PingPeriod)
	}
}

// kickSub removes sub from map
func (svc *UdpSvc) kickSub(conn *net.UDPConn, sub *Sub, raddr string) {
	svc.subsMu.Lock()
	delete(svc.subs, raddr)
	svc.subsMu.Unlock()
	fmt.Printf("kicked %s\n", raddr)
	svc.reply("kick", sub.raddr)
}
