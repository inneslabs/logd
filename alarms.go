/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package main

import (
	"fmt"
	"time"

	"github.com/intob/jfmt"
	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/cmd"
)

func prodErrors(slackWebhook string) *alarm.Alarm {
	a := &alarm.Alarm{
		Name: "Prod errors",
		Match: func(m *cmd.Msg) bool {
			if m.GetLvl() != cmd.Lvl_ERROR {
				return false
			}
			if m.GetEnv() != "prod" {
				return false
			}
			return true
		},
		Period:    time.Hour,
		Threshold: 100,
	}
	a.Action = func() error {
		top5 := alarm.GenerateTopNView(a.Report, 5)
		msg := fmt.Sprintf("We've had %d errors on prod in the last %s. Top 5:\n%s",
			a.EventCount.Load(),
			jfmt.FmtDuration(a.Period),
			top5)
		fmt.Println(msg)
		return alarm.SendSlackMsg(msg, slackWebhook)
	}
	return a
}
