package client

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	conn        net.Conn
	rateLimiter *rate.Limiter
}

type Cfg struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	RateLimitEvery time.Duration `yaml:"ratelimit_every"`
	RateLimitBurst int           `yaml:"ratelimit_burst"`
}

func NewClient(cfg *Cfg) (*Client, error) {
	var ip string
	if parsed := net.ParseIP(cfg.Host); parsed != nil {
		ip = fmt.Sprintf("[%s]", parsed.String())
	} else {
		addrs, err := net.LookupHost(cfg.Host)
		if err != nil {
			return nil, fmt.Errorf("error looking up host: %w", err)
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("no addresses found for host: %s", cfg.Host)
		}
		ip = addrs[0]
	}
	address := net.JoinHostPort(ip, strconv.Itoa(cfg.Port))
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, fmt.Errorf("error dialing: %w", err)
	}
	limit := rate.NewLimiter(rate.Every(cfg.RateLimitEvery), cfg.RateLimitBurst)
	return &Client{
		conn:        conn,
		rateLimiter: limit,
	}, nil
}
