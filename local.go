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
	c, err := conn.GetConn(addr)
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
		data, ok := m.([]byte)
		if ok {
			t.In <- &data
		}
	}
}
