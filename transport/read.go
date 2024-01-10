/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
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
			c := &cmd.Cmd{}
			err = proto.Unmarshal(payload, c)
			if err != nil {
				fmt.Println("protobuf unmarshal err:", err)
				continue
			}
			switch c.GetName() {
			case cmd.Name_WRITE:
				err = t.handleWrite(c, raddr, sum, timeBytes, payload)
			case cmd.Name_TAIL:
				err = t.handleTail(conn, raddr, sum, timeBytes, payload)
			case cmd.Name_PING:
				err = t.handlePing(raddr, sum, timeBytes, payload)
			}
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (t *Transporter) handleWrite(c *cmd.Cmd, raddr *net.UDPAddr, sum, timeBytes, payload []byte) error {
	if c.Msg == nil {
		return errors.New("msg is nil")
	}
	valid, err := auth.Verify(t.writeSecret, sum, timeBytes, payload)
	if !valid || err != nil {
		return fmt.Errorf("%s unauthorised to write: %w", raddr.IP.String(), err)
	}
	// pipe to tails
	t.Out <- &payload
	// pipe to alarm svc
	t.alarmSvc.In <- c.Msg
	// write to buffer
	t.buf.Write(&payload)
	return nil
}

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

func (t *Transporter) handlePing(raddr *net.UDPAddr, sum, timeBytes, payload []byte) error {
	valid, err := auth.Verify(t.readSecret, sum, timeBytes, payload)
	if !valid || err != nil {
		return fmt.Errorf("%s unauthorised to tail: %w", raddr.IP.String(), err)
	}
	t.mu.Lock()
	sub := t.subs[raddr.AddrPort().String()]
	if sub != nil {
		sub.lastPing = time.Now()
	}
	t.mu.Unlock()
	return nil
}
