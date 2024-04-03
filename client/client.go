package client

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/pkg"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
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
	var rateLimiter *rate.Limiter
	if cfg.RateLimitEvery > 0 {
		rateLimiter = rate.NewLimiter(
			rate.Every(cfg.RateLimitEvery),
			cfg.RateLimitBurst)
	}
	return &Client{conn, rateLimiter}, nil
}

func (cl *Client) SignCmd(ctx context.Context, command *cmd.Cmd, secret []byte) ([]byte, error) {
	payload, err := proto.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("err marshalling cmd: %w", err)
	}
	return pkg.Sign(secret, payload), nil
}

func (cl *Client) Wait(ctx context.Context) error {
	if cl.rateLimiter != nil {
		return cl.rateLimiter.Wait(ctx)
	}
	return nil
}

func (cl *Client) Write(signed []byte) error {
	_, err := cl.conn.Write(signed)
	if err != nil {
		return fmt.Errorf("err writing to socket: %w", err)
	}
	return nil
}
