package udp

import (
	"net/netip"
	"time"

	"github.com/inneslabs/logd/sign"
)

func (svc *UdpSvc) handlePing(raddr netip.AddrPort, pkg *sign.Pkg) {
	valid, err := svc.signer.Verify(svc.readSecret, pkg)
	if !valid || err != nil {
		return
	}
	if !svc.guard.Good(pkg.Sum) {
		return
	}
	svc.subsMu.Lock()
	sub := svc.subs[raddr.String()]
	if sub != nil {
		sub.lastPing = time.Now()
	}
	svc.subsMu.Unlock()
}
