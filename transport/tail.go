package transport

import (
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func (t *Transporter) handleTail(c *cmd.Cmd, raddrPort netip.AddrPort, unpk *auth.Unpacked) error {
	valid, err := auth.Verify(t.readSecret, unpk)
	if !valid || err != nil {
		return errors.New("unauthorized")
	}
	t.subsMu.Lock()
	t.subs[raddrPort.String()] = &Sub{
		raddrPort:   raddrPort,
		lastPing:    time.Now(),
		queryParams: c.GetQueryParams(),
	}
	t.subsMu.Unlock()
	txt := "tailing logs..."
	resp, _ := proto.Marshal(&cmd.Msg{
		Fn:  "logd",
		Lvl: cmd.Lvl_INFO.Enum(),
		Txt: &txt,
	})
	t.conn.WriteToUDPAddrPort(resp, raddrPort)
	fmt.Println("got new tail", raddrPort.String())
	return nil
}
