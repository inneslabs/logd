/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package main

import (
	"time"

	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/cmd"
)

func prodWpErrors(slackWebhook string) *alarm.Alarm {
	return &alarm.Alarm{
		Name: "prod/wp/error",
		Match: func(m *cmd.Msg) bool {
			if m.GetResponseStatus() != 200 {
				return false
			}
			if m.GetSvc() != "wp" {
				return false
			}
			if m.GetEnv() != "prod" {
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

func prodErrors(slackWebhook string) *alarm.Alarm {
	return &alarm.Alarm{
		Name: "prod/error",
		Match: func(m *cmd.Msg) bool {
			if m.GetLvl() != cmd.Lvl_ERROR {
				return false
			}
			if m.GetEnv() != "prod" {
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
