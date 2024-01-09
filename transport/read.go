package transport

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/msg"
	"google.golang.org/protobuf/proto"
)

func (t *Transporter) readFromConn(ctx context.Context, conn *net.UDPConn) {
	var buf []byte
	for {
		select {
		case <-ctx.Done():
			return
		default:
			buf = make([]byte, socketBufferSize)
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, raddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			sum, timeBytes, payload, err := auth.UnpackSignedMsg(buf[:n])
			if err != nil {
				fmt.Println("unpack msg err:", err)
				continue
			}
			payloadStr := string(payload)
			if payloadStr == "tail" || payloadStr == "ping" {
				valid, err := auth.Verify(t.readSecret, sum, timeBytes, payload)
				if !valid || err != nil {
					fmt.Printf("%s unauthorised: %s\r\n", raddr.IP.String(), err)
					continue
				}
				if string(payload) == "tail" {
					go t.handleTail(conn, raddr)
					continue
				}
				if string(payload) == "ping" {
					go t.handlePing(raddr)
					continue
				}
			}
			valid, err := auth.Verify(t.writeSecret, sum, timeBytes, payload)
			if !valid || err != nil {
				fmt.Printf("%s unauthorised: %s\r\n", raddr.IP.String(), err)
				continue
			}
			t.In <- &payload
		}
	}
}

func (t *Transporter) handleTail(conn *net.UDPConn, raddr *net.UDPAddr) {
	t.mu.Lock()
	t.subs[raddr.AddrPort().String()] = &Sub{
		raddr:    raddr,
		lastPing: time.Now(),
	}
	t.mu.Unlock()
	txt := "tailing logs..."
	payload, err := proto.Marshal(&msg.Msg{
		Fn:  "logd",
		Txt: &txt,
	})
	if err != nil {
		fmt.Println("pack msg err:", err)
		return
	}
	_, err = conn.WriteToUDP(payload, raddr)
	if err != nil {
		fmt.Printf("write udp err: (%s) %s\r\n", raddr, err)
		return
	}
	fmt.Println("got new tail", raddr.AddrPort().String())
}

func (t *Transporter) handlePing(raddr *net.UDPAddr) {
	t.mu.Lock()
	sub := t.subs[raddr.AddrPort().String()]
	if sub != nil {
		sub.lastPing = time.Now()
	}
	t.mu.Unlock()
}
