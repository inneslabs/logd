/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package query

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func QueryMsg(q *cmd.QueryParams, conn net.Conn, readSecret []byte) (<-chan *cmd.Msg, error) {
	err := writeRequest(q, conn, readSecret)
	if err != nil {
		return nil, err
	}
	out := make(chan *cmd.Msg)
	go readMsg(conn, out)
	return out, nil
}

func QueryBytes(q *cmd.QueryParams, conn net.Conn, readSecret []byte) (<-chan []byte, error) {
	err := writeRequest(q, conn, readSecret)
	if err != nil {
		return nil, err
	}
	out := make(chan []byte)
	go readBytes(conn, out)
	return out, nil
}

func writeRequest(q *cmd.QueryParams, conn net.Conn, readSecret []byte) error {
	payload, err := proto.Marshal(&cmd.Cmd{
		Name:        cmd.Name_QUERY,
		QueryParams: q,
	})
	if err != nil {
		return fmt.Errorf("marshal ping msg err: %w", err)
	}
	sig, err := auth.Sign(readSecret, payload, time.Now())
	if err != nil {
		return fmt.Errorf("sign tail msg err: %w", err)
	}
	_, err = conn.Write(sig)
	if err != nil {
		return fmt.Errorf("write tail msg err: %w", err)
	}
	return nil
}

func readMsg(conn net.Conn, out chan<- *cmd.Msg) {
	for {
		buf := make([]byte, 2048)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("error reading from conn: %s\r\n", err)
		}
		m := &cmd.Msg{}
		err = proto.Unmarshal(buf[:n], m)
		if err != nil {
			fmt.Println("unpack msg err:", err)
			continue
		}
		out <- m
	}
}

func readBytes(conn net.Conn, out chan<- []byte) {
	pool := &sync.Pool{
		New: func() interface{} {
			b := make([]byte, 2048)
			return &b
		},
	}
	for {
		bufPtr := pool.Get().(*[]byte)
		buf := *bufPtr
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("error reading from conn: %s\r\n", err)
		}
		out <- buf[:n]
		pool.Put(bufPtr)
	}
}
