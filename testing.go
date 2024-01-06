package main

import (
	"fmt"
	"os"

	"github.com/swissinfo-ch/logd/conn"
	"github.com/swissinfo-ch/logd/tail"
	"github.com/swissinfo-ch/logd/transport"
)

var (
	testTailHostname   = os.Getenv("TEST_TAIL_HOSTNAME")
	testTailReadSecret = os.Getenv("TEST_TAIL_READ_SECRET")
)

func tailForTesting(t *transport.Transporter) {
	addr, err := conn.GetAddr(testTailHostname)
	if err != nil {
		fmt.Println("failed to get addr:", err)
		return
	}
	c, err := conn.GetConn(addr)
	if err != nil {
		fmt.Println("failed to get conn:", err)
		return
	}
	msgs, err := tail.Tail(c, []byte(testTailReadSecret), tail.Plain)
	if err != nil {
		fmt.Println("failed to get tail:", err)
		return
	}
	for m := range msgs {
		data, ok := m.([]byte)
		if ok {
			t.In <- &data
		}
	}
}
