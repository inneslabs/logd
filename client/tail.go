package client

import (
	"context"
	"fmt"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/udp"
	"google.golang.org/protobuf/proto"
)

func (c *Client) Tail(ctx context.Context, q *cmd.QueryParams, secret []byte) (<-chan *cmd.Msg, error) {
	err := c.Cmd(ctx, &cmd.Cmd{
		Name:        cmd.Name_TAIL,
		QueryParams: q,
	}, secret)
	if err != nil {
		return nil, fmt.Errorf("err sending tail cmd: %w", err)
	}
	fmt.Printf("\rsent tail cmd\033[0K")
	out := make(chan *cmd.Msg)
	go c.readTailMsgs(out)
	go func() {
		for {
			time.Sleep(udp.PingPeriod)
			c.Cmd(ctx, &cmd.Cmd{
				Name: cmd.Name_PING,
			}, secret)
		}
	}()
	return out, nil
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
			continue
		}
		out <- m
	}
}
