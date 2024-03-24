package udp

import (
	"fmt"
	"time"
)

const (
	PingPeriod            = time.Second * 2
	kickAfterMissingPings = 5
)

func (svc *UdpSvc) kickLateSubs() {
	for {
		select {
		case <-time.After(PingPeriod):
			for _, sub := range svc.subs {
				if sub.lastPing.Before(time.Now().Add(-(PingPeriod * kickAfterMissingPings))) {
					svc.kickSub(sub)
					return
				}
			}
		case <-svc.ctx.Done():
			fmt.Println("kickLateSubs ended")
			return
		}
	}
}

// kickSub removes sub from map
func (svc *UdpSvc) kickSub(sub *Sub) {
	svc.subsMu.Lock()
	delete(svc.subs, sub.raddr.String())
	svc.subsMu.Unlock()
	fmt.Printf("kicked %s\n", sub.raddr.String())
	svc.reply("kick", sub.raddr)
}
