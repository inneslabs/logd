package client

import (
	"fmt"
	"time"

	"github.com/inneslabs/logd/auth"
	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/udp"
	"google.golang.org/protobuf/proto"
)

func (c *Client) Tail(q *cmd.QueryParams) (<-chan *cmd.Msg, error) {
	err := c.sendTailCmd(q)
	if err != nil {
		return nil, fmt.Errorf("send tail cmd err: %w", err)
	}
	fmt.Printf("\rsent tail cmd\033[0K")
	out := make(chan *cmd.Msg)
	go c.readTailMsgs(out)
	go c.ping()
	return out, nil
}

func (c *Client) sendTailCmd(q *cmd.QueryParams) error {
	payload, err := proto.Marshal(&cmd.Cmd{
		Name:        cmd.Name_TAIL,
		QueryParams: q,
	})
	if err != nil {
		return fmt.Errorf("marshal ping msg err: %w", err)
	}
	sig, err := auth.Sign(c.readSecret, payload, time.Now())
	if err != nil {
		return fmt.Errorf("sign tail msg err: %w", err)
	}
	_, err = c.conn.Write(sig)
	if err != nil {
		return fmt.Errorf("write tail msg err: %w", err)
	}
	return nil
}

func (c *Client) readTailMsgs(out chan<- *cmd.Msg) {
	buf := make([]byte, udp.MaxPacketSize)
	for {
		buf = buf[:udp.MaxPacketSize] // re-slice to capacity
		n, err := c.conn.Read(buf)
		if err != nil {
			fmt.Printf("\rerror reading from conn: %s\n", err)
		}
		m := &cmd.Msg{}
		err = proto.Unmarshal(buf[:n], m)
		if err != nil {
			fmt.Println("unpack msg err:", err)
			continue
		}
		if m.Key == udp.ReplyKey {
			fmt.Print(m.GetTxt())
		} else {
			out <- m
		}
	}
}

func (c *Client) ping() {
	for {
		time.Sleep(udp.PingPeriod)
		payload, err := proto.Marshal(&cmd.Cmd{
			Name: cmd.Name_PING,
		})
		if err != nil {
			fmt.Println("marshal ping msg err:", err)
			continue
		}
		sig, err := auth.Sign(c.readSecret, payload, time.Now())
		if err != nil {
			fmt.Println("sign ping msg err:", err)
			continue
		}
		_, err = c.conn.Write(sig)
		if err != nil {
			fmt.Println("write ping msg err:", err)
		}
	}
}
