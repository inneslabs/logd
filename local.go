package main

import (
	"fmt"

	"github.com/swissinfo-ch/logd/conn"
	"github.com/swissinfo-ch/logd/tail"
	"github.com/swissinfo-ch/logd/transport"
)

func tailLogd(t *transport.Transporter, tailHost, tailReadSecret string) {
	if tailHost == "" {
		return
	}
	addr, err := conn.GetAddr(tailHost)
	if err != nil {
		fmt.Println("failed to get addr:", err)
		return
	}
	c, err := conn.Dial(addr)
	if err != nil {
		fmt.Println("failed to get conn:", err)
		return
	}
	msgs, err := tail.Tail(c, []byte(tailReadSecret), tail.Plain)
	if err != nil {
		fmt.Println("failed to get tail:", err)
		return
	}
	fmt.Println("tailing", tailHost)
	for m := range msgs {
		payload, ok := m.([]byte)
		if ok {
			// pipe to tails
			t.Out <- &payload
			// write to buffer
			buf.Write(&payload)
			// don't pipe to alarm svc
			// because this would require unmarshaling the payload
			// only implement here if required for testing.
		}
	}
}
