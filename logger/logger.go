package logger

import (
	"context"
	"fmt"

	"github.com/intob/logd/client"
	"github.com/intob/logd/cmd"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Logger struct {
	ctx    context.Context
	client *client.Client
	secret []byte
	msgKey string
	stdout bool
}

type LoggerCfg struct {
	Client *client.Cfg `yaml:"client"`
	Secret string      `yaml:"secret"`
	MsgKey string      `yaml:"msg_key"`
	Stdout bool        `yaml:"stdout"`
}

func NewLogger(ctx context.Context, cfg *LoggerCfg) (*Logger, error) {
	cl, err := client.NewClient(cfg.Client)
	if err != nil {
		return nil, fmt.Errorf("err initializing client: %w", err)
	}
	return &Logger{
		ctx:    ctx,
		client: cl,
		secret: []byte(cfg.Secret),
		msgKey: cfg.MsgKey,
		stdout: cfg.Stdout,
	}, nil
}

func (l *Logger) Log(lvl cmd.Lvl, template string, args ...interface{}) {
	txt := fmt.Sprintf(template, args...)
	signed, _ := l.client.SignCmd(l.ctx, &cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:   timestamppb.Now(),
			Key: l.msgKey,
			Lvl: lvl,
			Txt: txt,
		},
	}, l.secret)
	l.client.Wait(l.ctx)
	l.client.Write(signed)
	if l.stdout {
		fmt.Printf(template+"\n", args...)
	}
}

func (l *Logger) Error(template string, args ...interface{}) {
	l.Log(cmd.Lvl_ERROR, template, args...)
}

func (l *Logger) Warn(template string, args ...interface{}) {
	l.Log(cmd.Lvl_WARN, template, args...)
}

func (l *Logger) Info(template string, args ...interface{}) {
	l.Log(cmd.Lvl_INFO, template, args...)
}

func (l *Logger) Debug(template string, args ...interface{}) {
	l.Log(cmd.Lvl_DEBUG, template, args...)
}

func (l *Logger) Trace(template string, args ...interface{}) {
	l.Log(cmd.Lvl_TRACE, template, args...)
}
