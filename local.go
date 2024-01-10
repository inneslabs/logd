/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
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
	msgs, err := tail.TailPlain(c, []byte(tailReadSecret))
	if err != nil {
		fmt.Println("failed to get tail:", err)
		return
	}
	fmt.Println("tailing", tailHost)
	for m := range msgs {
		// pipe to tails
		t.Out <- m
		// write to buffer
		buf.Write(m)
		// don't pipe to alarm svc
		// because this would require unmarshaling the payload
		// only implement if later required.
	}
}
