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
	Conn   net.Conn
	Secret []byte
	Env    string
	Svc    string
	Fn     string
}

func NewLogger(host, env, svc, fn string, writeSecret []byte) (*Logger, error) {
	addr, err := conn.GetAddr(host)
	if err != nil {
		return nil, fmt.Errorf("get addr err: %w", err)
	}
	c, err := conn.GetConn(addr)
	if err != nil {
		return nil, fmt.Errorf("get conn err: %w", err)
	}
	return &Logger{
		Conn:   c,
		Secret: writeSecret,
		Env:    env,
		Svc:    svc,
		Fn:     fn,
	}, nil
}

// Log writes a logd entry to Logger Conn
func (l *Logger) Log(lvl Lvl, template string, args ...interface{}) {
	// build msg
	payload, err := pack.PackMsg(&msg.Msg{
		Timestamp: time.Now().UnixNano(),
		Env:       l.Env,
		Svc:       l.Svc,
		Fn:        l.Fn,
		Lvl:       string(lvl),
		Msg:       fmt.Sprintf(template, args...),
	})
	if err != nil {
		fmt.Println("logd.log pack msg err:", err)
		return
	}

	// get ephemeral signature
	signedMsg, err := auth.Sign(l.Secret, payload, time.Now())
	if err != nil {
		fmt.Println("logd.log sign msg err:", err)
		return
	}

	// write to socket
	_, err = l.Conn.Write(signedMsg)
	if err != nil {
		fmt.Println("logd.log write udp err:", err)
	}
}
