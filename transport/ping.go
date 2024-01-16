package transport

import (
	"fmt"
	"net"
	"time"

	"github.com/swissinfo-ch/logd/auth"
)

func (t *Transporter) handlePing(raddr *net.UDPAddr, sum, timeBytes, payload []byte) error {
	valid, err := auth.Verify(t.readSecret, sum, timeBytes, payload)
	if !valid || err != nil {
		return fmt.Errorf("%s unauthorised to tail: %w", raddr.IP.String(), err)
	}
	t.subsMu.Lock()
	sub := t.subs[raddr.AddrPort().String()]
	if sub != nil {
		sub.lastPing = time.Now()
	}
	t.subsMu.Unlock()
	return nil
}
