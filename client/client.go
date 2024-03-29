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
	var ip string
	if parsed := net.ParseIP(cfg.Host); parsed != nil {
		ip = fmt.Sprintf("[%s]", parsed.String())
	} else {
		addrs, err := net.LookupHost(cfg.Host)
		if err != nil {
			return nil, fmt.Errorf("error looking up hostname: %w", err)
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("no addresses found for hostname: %s", cfg.Host)
		}
		ip = addrs[0]
	}
	address := net.JoinHostPort(ip, strconv.Itoa(cfg.Port))
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, fmt.Errorf("error dialing: %w", err)
	}
	return &Client{[]byte(cfg.ReadSecret), conn}, nil
}
