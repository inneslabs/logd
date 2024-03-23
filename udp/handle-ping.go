package udp

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/intob/logd/auth"
)

func (svc *UdpSvc) handlePing(raddr netip.AddrPort, unpk *auth.Unpacked) error {
	valid, err := auth.Verify(svc.readSecret, unpk)
	if !valid || err != nil {
		return fmt.Errorf("%s unauthorised to tail: %w", raddr.String(), err)
	}
	svc.subsMu.Lock()
	sub := svc.subs[raddr.String()]
	if sub != nil {
		sub.lastPing = time.Now()
	}
	svc.subsMu.Unlock()
	return nil
}
