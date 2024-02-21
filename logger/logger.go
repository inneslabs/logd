/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package logger

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Logger struct {
	ctx         context.Context
	conn        net.Conn
	rateLimiter *rate.Limiter
	secret      []byte
	env         string
	svc         string
	fn          string
	stdout      bool
}

type LoggerConfig struct {
	Ctx         context.Context
	Host        string
	Port        int
	WriteSecret string
	Env         string
	Svc         string
	Fn          string
	Stdout      bool
}

// Returns a new logger, defaults to logd.swissinfo.ch:6102
func NewLogger(cfg *LoggerConfig) (*Logger, error) {
	if cfg.Host == "" {
		cfg.Host = "logd.swissinfo.ch"
	}
	if cfg.Port == 0 {
		cfg.Port = 6102
	}
	addrs, err := net.LookupHost(cfg.Host)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial("udp", addrs[0]+":"+strconv.Itoa(cfg.Port))
	if err != nil {
		return nil, err
	}
	return &Logger{
		ctx:         cfg.Ctx,
		conn:        conn,
		rateLimiter: rate.NewLimiter(rate.Every(time.Microsecond*250), 20),
		secret:      []byte(cfg.WriteSecret),
		env:         cfg.Env,
		svc:         cfg.Svc,
		fn:          cfg.Fn,
		stdout:      cfg.Stdout,
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
	l.rateLimiter.Wait(l.ctx)
	l.conn.Write(signedMsg)
	if l.stdout {
		fmt.Printf(template+"\n", args...)
	}
}

func (l *Logger) Error(template string, args ...interface{}) {
	l.Log(cmd.Lvl_ERROR.Enum(), template, args...)
}

func (l *Logger) Warn(template string, args ...interface{}) {
	l.Log(cmd.Lvl_WARN.Enum(), template, args...)
}

func (l *Logger) Info(template string, args ...interface{}) {
	l.Log(cmd.Lvl_INFO.Enum(), template, args...)
}

func (l *Logger) Debug(template string, args ...interface{}) {
	l.Log(cmd.Lvl_DEBUG.Enum(), template, args...)
}

func (l *Logger) Trace(template string, args ...interface{}) {
	l.Log(cmd.Lvl_TRACE.Enum(), template, args...)
}
