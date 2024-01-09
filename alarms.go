package main

import (
	"time"

	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/msg"
)

func prodWpErrors() *alarm.Alarm {
	return &alarm.Alarm{
		Name: "prod/wp/error",
		Match: func(m *msg.Msg) bool {
			if m.Env != "prod" {
				return false
			}
			if m.Svc != "wp" {
				return false
			}
			if m.ResponseStatus == nil {
				return false
			}
			if *m.ResponseStatus != 200 {
				return false
			}
			return true
		},
		Period:    time.Minute * 10,
		Threshold: 10,
		Action: func() error {
			return alarm.SendSlackMsg("💥 We've had 10 errors on prod/wp in the last 10 minutes.", slackWebhook)
		},
	}
}

func prodErrors() *alarm.Alarm {
	return &alarm.Alarm{
		Name: "prod/error",
		Match: func(m *msg.Msg) bool {
			if m.Env != "prod" {
				return false
			}
			if m.Lvl == nil {
				return false
			}
			if *m.Lvl != msg.Lvl_ERROR {
				return false
			}
			return true
		},
		Period:    time.Minute * 10,
		Threshold: 50,
		Action: func() error {
			return alarm.SendSlackMsg("💣 We've had 50 errors on prod in the last 10 minutes.", slackWebhook)
		},
	}
}
