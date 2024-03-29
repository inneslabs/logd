package logger

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/inneslabs/logd/auth"
	"github.com/inneslabs/logd/cmd"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Logger struct {
	ctx         context.Context
	conn        net.Conn
	rateLimiter *rate.Limiter
	secret      []byte
	msgKey      string
	stdout      bool
}

type LoggerCfg struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Secret string `yaml:"secret"`
	MsgKey string `yaml:"msg_key"`
	Stdout bool   `yaml:"stdout"`
}

func NewLogger(ctx context.Context, cfg *LoggerCfg) (*Logger, error) {
	addrs, err := net.LookupHost(cfg.Host)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial("udp", addrs[0]+":"+strconv.Itoa(cfg.Port))
	if err != nil {
		return nil, err
	}
	return &Logger{
		ctx:         ctx,
		conn:        conn,
		rateLimiter: rate.NewLimiter(rate.Every(time.Microsecond*250), 20),
		secret:      []byte(cfg.Secret),
		msgKey:      cfg.MsgKey,
		stdout:      cfg.Stdout,
	}, nil
}

func (l *Logger) Log(lvl *cmd.Lvl, template string, args ...interface{}) {
	txt := fmt.Sprintf(template, args...)
	payload, _ := proto.Marshal(&cmd.Msg{
		T:   timestamppb.Now(),
		Key: l.msgKey,
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
