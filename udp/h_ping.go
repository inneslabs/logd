package udp

import (
	"net/netip"
	"time"

	"github.com/inneslabs/logd/pkg"
)

func (svc *UdpSvc) handlePing(raddr netip.AddrPort, p *pkg.Pkg) {
	if !svc.guard.Good(svc.readSecret, p) {
		return
	}
	svc.subsMu.Lock()
	sub := svc.subs[raddr.String()]
	if sub != nil {
		sub.lastPing = time.Now()
	}
	svc.subsMu.Unlock()
}
