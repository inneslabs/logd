package client

import (
	"context"
	"fmt"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/udp"
	"google.golang.org/protobuf/proto"
)

func (cl *Client) Tail(ctx context.Context, q *cmd.QueryParams, secret []byte) (<-chan *cmd.Msg, error) {
	signed, err := SignCmd(ctx, &cmd.Cmd{
		Name:        cmd.Name_TAIL,
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
	fmt.Printf("\rsent tail cmd\033[0K")
	out := make(chan *cmd.Msg)
	go cl.readTailMsgs(out)
	go cl.ping(ctx, secret)
	return out, nil
}

func (cl *Client) ping(ctx context.Context, secret []byte) {
	for {
		time.Sleep(udp.PingPeriod)
		signed, err := SignCmd(ctx, &cmd.Cmd{
			Name: cmd.Name_PING,
		}, secret)
		if err != nil {
			panic(err)
		}
		cl.Wait(ctx)
		err = cl.Write(signed)
		if err != nil {
			panic(err)
		}
	}
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
