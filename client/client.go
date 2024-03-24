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
	hostname   string
	port       int
	conn       net.Conn
}

type Cfg struct {
	Ctx        context.Context
	ReadSecret string
	Hostname   string
	Port       int
}

func NewClient(cfg *Cfg) (*Client, error) {
	addrs, err := net.LookupHost(cfg.Hostname)
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
		hostname:   cfg.Hostname,
		port:       cfg.Port,
		conn:       conn,
	}, nil
}
