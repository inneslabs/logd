package transport

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/swissinfo-ch/logd/auth"
)

func (t *Transporter) handlePing(raddrPort netip.AddrPort, unpk *auth.Unpacked) error {
	valid, err := auth.Verify(t.readSecret, unpk)
	if !valid || err != nil {
		return fmt.Errorf("%s unauthorised to tail: %w", raddrPort.String(), err)
	}
	t.subsMu.Lock()
	sub := t.subs[raddrPort.String()]
	if sub != nil {
		sub.lastPing = time.Now()
	}
	t.subsMu.Unlock()
	return nil
}
