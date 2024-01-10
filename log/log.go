/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package log

import (
	"fmt"
	"net"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"github.com/swissinfo-ch/logd/conn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

// Log writes a msg to Logger socket
func (l *Logger) Log(lvl *cmd.Lvl, template string, args ...interface{}) {
	txt := fmt.Sprintf(template, args...)
	payload, _ := proto.Marshal(&cmd.Msg{
		T:   timestamppb.Now(),
		Env: l.env,
		Svc: l.svc,
		Fn:  l.fn,
		Lvl: lvl,
		Txt: &txt,
	})
	signedMsg, _ := auth.Sign(l.secret, payload, time.Now())
	l.conn.Write(signedMsg)
}
