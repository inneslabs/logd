package udp

import (
	"net/netip"
	"time"

	"github.com/inneslabs/logd/sign"
)

func (svc *UdpSvc) handlePing(raddr netip.AddrPort, pkg *sign.Pkg) {
	if !svc.guard.Good(svc.readSecret, pkg) {
		return
	}
	svc.subsMu.Lock()
	sub := svc.subs[raddr.String()]
	if sub != nil {
		sub.lastPing = time.Now()
	}
	svc.subsMu.Unlock()
}
