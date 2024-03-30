package udp

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/sign"
)

func (svc *UdpSvc) handleTail(c *cmd.Cmd, raddr netip.AddrPort, pkg *sign.Pkg) {
	valid, err := svc.signer.Verify(svc.readSecret, pkg)
	if !valid || err != nil {
		return
	}
	if !svc.guard.Good(pkg.Sum) {
		return
	}
	svc.subsMu.Lock()
	svc.subs[raddr.String()] = &Sub{
		raddr:       raddr,
		lastPing:    time.Now(),
		queryParams: c.GetQueryParams(),
	}
	svc.subsMu.Unlock()
	svc.reply("\rtailing logs\033[0K", raddr)
	fmt.Println("got new tail", raddr.String())
}
