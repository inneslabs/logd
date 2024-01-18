package transport

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func (t *Transporter) handleTail(c *cmd.Cmd, conn *net.UDPConn, raddrPort netip.AddrPort, sum, timeBytes, payload []byte) error {
	valid, err := auth.Verify(t.readSecret, sum, timeBytes, payload)
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
	payload, _ = proto.Marshal(&cmd.Msg{
		Fn:  "logd",
		Lvl: cmd.Lvl_INFO.Enum(),
		Txt: &txt,
	})
	conn.WriteToUDPAddrPort(payload, raddrPort)
	fmt.Println("got new tail", raddrPort.String())
	return nil
}
