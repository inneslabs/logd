package udp

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/pkg"
)

func (svc *UdpSvc) handleTail(c *cmd.Cmd, raddr netip.AddrPort, p *pkg.Pkg) {
	if !svc.guard.Good([]byte(svc.secrets.Read), p) {
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
