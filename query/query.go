/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package query

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func Query(q *cmd.QueryParams, conn net.Conn, readSecret []byte) (<-chan *cmd.Msg, error) {
	err := writeRequest(q, conn, readSecret)
	if err != nil {
		return nil, err
	}
	out := make(chan *cmd.Msg)
	go readMsgs(conn, out)
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

func readMsgs(conn net.Conn, out chan<- *cmd.Msg) {
	pool := &sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
	for {
		m, err := readMsg(pool, conn)
		if err != nil {
			fmt.Println("failed to read msg:", err)
			continue
		}
		out <- m
	}
}

func readMsg(pool *sync.Pool, conn net.Conn) (*cmd.Msg, error) {
	buf := pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer pool.Put(buf)
	_, err := buf.ReadFrom(conn)
	if err != nil {
		return nil, err
	}
	m := &cmd.Msg{}
	err = proto.Unmarshal(buf.Bytes(), m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
