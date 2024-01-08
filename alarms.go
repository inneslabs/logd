package main

import (
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/msg"
	"github.com/swissinfo-ch/logd/wp"
)

func prodWpErrors() *alarm.Alarm {
	return &alarm.Alarm{
		Name: "prod/wp/error",
		Match: func(m *msg.Msg) bool {
			if m.Env != "prod" {
				return false
			}
			if m.Lvl != "ERROR" {
				return false
			}
			reqData, ok := m.Dump.([]byte)
			if !ok {
				fmt.Printf("dump is not a []byte, it is type %T\r\n", m.Dump)
				return false
			}
			req := &wp.Request{}
			err := cbor.Unmarshal(reqData, req)
			if err != nil {
				fmt.Println("failed to unmarshal dump:", err)
			}
			if req.ResponseStatus == "200" {
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
			if m.Lvl != "ERROR" {
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
