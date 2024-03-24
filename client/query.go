package client

import (
	"fmt"
	"time"

	"github.com/inneslabs/logd/auth"
	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/udp"
	"google.golang.org/protobuf/proto"
)

func (c *Client) Query(q *cmd.QueryParams) (<-chan *cmd.Msg, error) {
	err := c.writeRequest(q)
	if err != nil {
		return nil, err
	}
	out := make(chan *cmd.Msg)
	go c.readQueryMsgs(out)
	return out, nil
}

func (c *Client) writeRequest(q *cmd.QueryParams) error {
	payload, err := proto.Marshal(&cmd.Cmd{
		Name:        cmd.Name_QUERY,
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

func (c *Client) readQueryMsgs(out chan<- *cmd.Msg) {
	buf := make([]byte, udp.MaxPacketSize)
	for {
		m, err := c.readQueryMsg(buf)
		if err != nil {
			fmt.Println("failed to read msg:", err)
			continue
		}
		if m.GetKey() == udp.ReplyKey {
			if m.GetTxt() == udp.EndMsg {
				return
			}
			continue
		}
		out <- m
	}
}

func (c *Client) readQueryMsg(buf []byte) (*cmd.Msg, error) {
	buf = buf[:udp.MaxPacketSize] // re-slice to capacity
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	m := &cmd.Msg{}
	err = proto.Unmarshal(buf[:n], m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
