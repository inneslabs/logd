package transport

import (
	"context"
	"fmt"
	"net"
)

func (t *Transporter) writeToConn(ctx context.Context, conn *net.UDPConn) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-t.Out:
			for raddr, sub := range t.subs {
				go func(msg *[]byte, sub *Sub, raddr string) {
					_, err := conn.WriteToUDP(*msg, sub.raddr)
					if err != nil {
						fmt.Printf("write udp err: (%s) %s\r\n", raddr, err)
					}
				}(msg, sub, raddr)
			}
		}
	}
}
