/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/intob/jfmt"
	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/app"
	"github.com/swissinfo-ch/logd/store"
	"github.com/swissinfo-ch/logd/udp"
)

func main() {
	// no default, can also be blank
	readSecret := os.Getenv("LOGD_READ_SECRET")
	writeSecret := os.Getenv("LOGD_WRITE_SECRET")
	slackWebhook := os.Getenv("LOGD_SLACK_WEBHOOK")

	// defaults
	udpLaddrPort := ":6102" // string supports fly-global-services:6102
	appPort := 6101
	bufferSize := 1000000
	accessControlAllowOrigin := "*"

	udpPortEnv, set := os.LookupEnv("LOGD_UDP_LADDRPORT")
	if set {
		udpLaddrPort = udpPortEnv
	}

	appPortEnv, set := os.LookupEnv("LOGD_APP_PORT")
	if set {
		var err error
		appPort, err = strconv.Atoi(appPortEnv)
		if err != nil {
			panic("LOGD_APP_PORT must be an int")
		}
	}

	bufferSizeEnv, set := os.LookupEnv("LOGD_BUFFER_SIZE")
	if set {
		var err error
		bufferSize, err = strconv.Atoi(bufferSizeEnv)
		if err != nil {
			panic("LOGD_BUFFER_SIZE must be an int")
		}
	}

	accessControlAllowOriginEnv, set := os.LookupEnv("LOGD_ACCESS_CONTROL_ALLOW_ORIGIN")
	if set {
		accessControlAllowOrigin = accessControlAllowOriginEnv
	}

	// print config insensitive config
	fmt.Println("udp port set to", udpLaddrPort)
	fmt.Println("app port set to", appPort)
	fmt.Println("buffer size set to", bufferSize)
	fmt.Println("access-control-allow-origin set to", accessControlAllowOrigin)

	// init store
	svcSize := uint32(100000) // 100K logs per env/svc
	logStore := store.NewStore(&store.Cfg{
		RingSizes: map[string]uint32{
			"/prod/taxonomy-service": svcSize,
			"/prod/ticker-service":   svcSize,
			"/prod/logs":             svcSize,
			"/prod/swiplus-service":  svcSize,
			"/prod/video-service":    svcSize,
			"/prod/swi-core":         1000,
		},
		FallbackSize: 500000, // 500K as fallback
	})

	// init alarms
	alarmSvc := alarm.NewSvc()
	alarmSvc.Set(prodErrors10Min(slackWebhook))
	alarmSvc.Set(prodErrorsHourly(slackWebhook))

	// init root context
	ctx := getCtx()

	// init udp
	udp.NewSvc(&udp.Cfg{
		Ctx:                 ctx,
		LaddrPort:           udpLaddrPort,
		ReadSecret:          readSecret,
		WriteSecret:         writeSecret,
		LogStore:            logStore,
		AlarmSvc:            alarmSvc,
		SubRateLimitEvery:   100 * time.Microsecond,
		SubRateLimitBurst:   50,
		QueryRateLimitEvery: 20 * time.Millisecond,
		QueryRateLimitBurst: 10,
	})

	// init app
	app.NewApp(&app.Cfg{
		Ctx:                      ctx,
		LogStore:                 logStore,
		AlarmSvc:                 alarmSvc,
		RateLimitEvery:           time.Second,
		RateLimitBurst:           10,
		Port:                     appPort,
		AccessControlAllowOrigin: accessControlAllowOrigin,
	})

	// wait for kill signal
	<-ctx.Done()
	fmt.Println("all routines ended")
}

// prodErrors10Min returns an alarm that triggers on 10K prod errors in 10 minutes
func prodErrors10Min(slackWebhook string) *alarm.Alarm {
	// build alarm
	a := &alarm.Alarm{
		Name:      "10K prod errors in 10 minutes",
		Period:    10 * time.Minute,
		Threshold: 10000,
	}
	a.Action = func() error {
		top5 := alarm.GenerateTopNView(a.Report, 5)
		msg := fmt.Sprintf("%s: We've had %s errors on prod in the last %s.\n\nTop 5 errors:\n%s",
			a.Name,
			jfmt.FmtCount32(uint32(a.EventCount.Load())),
			jfmt.FmtDuration(a.Period),
			top5)
		fmt.Println(msg)
		return alarm.SendSlackMsg(msg, slackWebhook)
	}
	return a
}

// prodErrors returns an alarm that triggers on prod errors hourly
func prodErrorsHourly(slackWebhook string) *alarm.Alarm {
	// build alarm
	a := &alarm.Alarm{
		Name:      "Prod errors hourly",
		Period:    time.Hour,
		Threshold: 1,
	}
	a.Action = func() error {
		top5 := alarm.GenerateTopNView(a.Report, 5)
		msg := fmt.Sprintf("%s: We've had %s errors on prod in the last %s.\n\nTop 5 errors:\n%s",
			a.Name,
			jfmt.FmtCount32(uint32(a.EventCount.Load())),
			jfmt.FmtDuration(a.Period),
			top5)
		fmt.Println(msg)
		return alarm.SendSlackMsg(msg, slackWebhook)
	}
	return a
}

// cancelOnKillSig cancels the context on os interrupt kill signal
func cancelOnKillSig(sigs chan os.Signal, cancel context.CancelFunc) {
	switch <-sigs {
	case syscall.SIGINT:
		fmt.Println("\nreceived SIGINT")
	case syscall.SIGTERM:
		fmt.Println("\nreceived SIGTERM")
	}
	cancel()
}

// getCtx returns a root context that awaits a kill signal from os
func getCtx() context.Context {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go cancelOnKillSig(sigs, cancel)
	return ctx
}
