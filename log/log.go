package log

import (
	"fmt"
	"net"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/conn"
	"github.com/swissinfo-ch/logd/msg"
	"github.com/swissinfo-ch/logd/pack"
)

const (
	Error = Lvl("ERROR")
	Warn  = Lvl("WARN")
	Info  = Lvl("INFO")
	Debug = Lvl("DEBUG")
	Trace = Lvl("TRACE")
)

type Lvl string

type Logger struct {
	conn   net.Conn
	secret []byte
	env    string
	svc    string
	fn     string
}

type LoggerConfig struct {
	Host        string
	WriteSecret string
	Env         string
	Svc         string
	Fn          string
}

func NewLogger(cfg *LoggerConfig) (*Logger, error) {
	addr, err := conn.GetAddr(cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("get addr err: %w", err)
	}
	c, err := conn.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("get conn err: %w", err)
	}
	return &Logger{
		conn:   c,
		secret: []byte(cfg.WriteSecret),
		env:    cfg.Env,
		svc:    cfg.Svc,
		fn:     cfg.Fn,
	}, nil
}

// Log writes a logd entry to Logger Conn
func (l *Logger) Log(lvl Lvl, template string, args ...interface{}) {
	// build msg
	payload, err := pack.PackMsg(&msg.Msg{
		Timestamp: time.Now().UnixNano(),
		Env:       l.env,
		Svc:       l.svc,
		Fn:        l.fn,
		Lvl:       string(lvl),
		Msg:       fmt.Sprintf(template, args...),
	})
	if err != nil {
		fmt.Println("logd.log pack msg err:", err)
		return
	}

	// get ephemeral signature
	signedMsg, err := auth.Sign(l.secret, payload, time.Now())
	if err != nil {
		fmt.Println("logd.log sign msg err:", err)
		return
	}

	// write to socket
	_, err = l.conn.Write(signedMsg)
	if err != nil {
		fmt.Println("logd.log write udp err:", err)
	}
}
