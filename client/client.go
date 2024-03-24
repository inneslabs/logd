package client

import (
	"fmt"
	"net"
	"strconv"
)

type Client struct {
	readSecret []byte
	conn       net.Conn
}

type Cfg struct {
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
	return &Client{[]byte(cfg.ReadSecret), conn}, nil
}
