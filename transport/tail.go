package transport

import (
	"fmt"
	"net"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func (t *Transporter) handleTail(conn *net.UDPConn, raddr *net.UDPAddr, sum, timeBytes, payload []byte) error {
	valid, err := auth.Verify(t.readSecret, sum, timeBytes, payload)
	if !valid || err != nil {
		return fmt.Errorf("%s unauthorised to tail: %w", raddr.IP.String(), err)
	}
	t.mu.Lock()
	t.subs[raddr.AddrPort().String()] = &Sub{
		raddr:    raddr,
		lastPing: time.Now(),
	}
	t.mu.Unlock()
	txt := "tailing logs..."
	payload, err = proto.Marshal(&cmd.Msg{
		Fn:  "logd",
		Lvl: cmd.Lvl_INFO.Enum(),
		Txt: &txt,
	})
	if err != nil {
		return fmt.Errorf("protobuf marshal err: %w", err)
	}
	_, err = conn.WriteToUDP(payload, raddr)
	if err != nil {
		return fmt.Errorf("write udp err: (%s) %s", raddr, err)
	}
	fmt.Println("got new tail", raddr.AddrPort().String())
	return nil
}
