package udp

import (
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
)

func (svc *UdpSvc) handleTail(c *cmd.Cmd, raddrPort netip.AddrPort, unpk *auth.Unpacked) error {
	valid, err := auth.Verify(svc.readSecret, unpk)
	if !valid || err != nil {
		return errors.New("unauthorized")
	}
	svc.subsMu.Lock()
	svc.subs[raddrPort.String()] = &Sub{
		raddrPort:   raddrPort,
		lastPing:    time.Now(),
		queryParams: c.GetQueryParams(),
	}
	svc.subsMu.Unlock()
	svc.reply("tailing logs", raddrPort)
	fmt.Println("got new tail", raddrPort.String())
	return nil
}
