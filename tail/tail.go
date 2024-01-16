/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package tail

import (
	"fmt"
	"net"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"github.com/swissinfo-ch/logd/transport"
	"google.golang.org/protobuf/proto"
)

func TailMsg(q *cmd.QueryParams, conn net.Conn, readSecret []byte) (<-chan *cmd.Msg, error) {
	err := sendTailCmd(q, conn, readSecret)
	if err != nil {
		return nil, fmt.Errorf("send tail cmd err: %w", err)
	}
	out := make(chan *cmd.Msg)
	go readMsg(conn, out)
	go ping(conn, readSecret)
	return out, nil
}

func TailPlain(q *cmd.QueryParams, conn net.Conn, readSecret []byte) (<-chan []byte, error) {
	err := sendTailCmd(q, conn, readSecret)
	if err != nil {
		return nil, fmt.Errorf("send tail cmd err: %w", err)
	}
	out := make(chan []byte)
	go readPlain(conn, out)
	go ping(conn, readSecret)
	return out, nil
}

func sendTailCmd(q *cmd.QueryParams, conn net.Conn, readSecret []byte) error {
	payload, err := proto.Marshal(&cmd.Cmd{
		Name:        cmd.Name_TAIL,
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

func readPlain(conn net.Conn, out chan<- []byte) {
	for {
		buf := make([]byte, 2048)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("error reading from conn: %s\r\n", err)
		}
		out <- buf[:n]
	}
}

func ping(conn net.Conn, readSecret []byte) {
	for {
		time.Sleep(transport.PingPeriod)
		payload, err := proto.Marshal(&cmd.Cmd{
			Name: cmd.Name_PING,
		})
		if err != nil {
			fmt.Println("marshal ping msg err:", err)
			continue
		}
		sig, err := auth.Sign(readSecret, payload, time.Now())
		if err != nil {
			fmt.Println("sign ping msg err:", err)
			continue
		}
		_, err = conn.Write(sig)
		if err != nil {
			fmt.Println("write ping msg err:", err)
		}
	}
}
