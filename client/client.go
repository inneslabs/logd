package client

import (
	"context"
	"fmt"
	"net"
	"strconv"
)

type Client struct {
	ctx        context.Context
	readSecret []byte
	conn       net.Conn
}

type Cfg struct {
	Ctx        context.Context
	ReadSecret string
	Host       string
	Port       int
}

func NewClient(cfg *Cfg) (*Client, error) {
	addrs, err := net.LookupHost(cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("err looking up hostname: %w", err)
	}

	conn, err := net.Dial("udp", addrs[0]+":"+strconv.Itoa(cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("err dialing: %w", err)
	}

	return &Client{
		ctx:        cfg.Ctx,
		readSecret: []byte(cfg.ReadSecret),
		conn:       conn,
	}, nil
}
