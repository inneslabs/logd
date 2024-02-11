package udp

import (
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
)

func (svc *UdpSvc) handleTail(c *cmd.Cmd, raddr netip.AddrPort, unpk *auth.Unpacked) error {
	valid, err := auth.Verify(svc.readSecret, unpk)
	if !valid || err != nil {
		return errors.New("unauthorized")
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
	return nil
}
