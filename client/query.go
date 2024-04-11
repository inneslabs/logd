package client

import (
	"context"
	"fmt"
	"time"

	"github.com/intob/logd/cmd"
	"github.com/intob/logd/udp"
	"google.golang.org/protobuf/proto"
)

func (cl *Client) Query(ctx context.Context, q *cmd.QueryParams, secret []byte) (<-chan *cmd.Msg, error) {
	signed, err := cl.SignCmd(ctx, &cmd.Cmd{
		Name:        cmd.Name_QUERY,
		QueryParams: q,
	}, secret)
	if err != nil {
		return nil, err
	}
	err = cl.Wait(ctx)
	if err != nil {
		return nil, err
	}
	err = cl.Write(signed)
	if err != nil {
		return nil, err
	}
	out := make(chan *cmd.Msg)
	go cl.readQueryMsgs(out)
	return out, nil
}

func (c *Client) readQueryMsgs(out chan<- *cmd.Msg) {
	defer close(out)
	buf := make([]byte, c.packetBufferSize)
	for {
		m, err := c.readQueryMsg(buf)
		if err != nil {
			fmt.Println("failed to read msg:", err)
			return
		}
		if m.Key == udp.ReplyKey && m.Txt == udp.EndMsg {
			return
		}
		out <- m
	}
}

func (c *Client) readQueryMsg(buf []byte) (*cmd.Msg, error) {
	buf = buf[:c.packetBufferSize] // re-slice to capacity
	deadline := time.Now().Add(500 * time.Millisecond)
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return nil, err
	}
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	c.conn.SetReadDeadline(time.Time{})
	m := &cmd.Msg{}
	err = proto.Unmarshal(buf[:n], m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
