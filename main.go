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

	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/app"
	"github.com/swissinfo-ch/logd/ring"
	"github.com/swissinfo-ch/logd/udp"
)

func main() {
	// no default, can also be blank
	readSecret := os.Getenv("LOGD_READ_SECRET")
	writeSecret := os.Getenv("LOGD_WRITE_SECRET")
	accessControlAllowOrigin := os.Getenv("LOGD_ACCESS_CONTROL_ALLOW_ORIGIN")
	slackWebhook := os.Getenv("LOGD_SLACK_WEBHOOK")

	// defaults
	udpLaddrPort := ":6102" // string supports fly-global-services:6102
	appPort := 6101
	bufferSize := 1000000

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

	fmt.Println("udp port set to", udpLaddrPort)
	fmt.Println("app port set to", appPort)
	fmt.Println("buffer size set to", bufferSize)

	// init ring buffer
	ringBuf := ring.NewRingBuffer(uint32(bufferSize))

	// init alarms
	alarmSvc := alarm.NewSvc()
	alarmSvc.Set(prodWpErrors(slackWebhook))
	alarmSvc.Set(prodErrors(slackWebhook))

	// init root context
	ctx := getCtx()

	// init udp
	udp.NewSvc(&udp.Cfg{
		Ctx:                 ctx,
		LaddrPort:           udpLaddrPort,
		ReadSecret:          readSecret,
		WriteSecret:         writeSecret,
		RingBuf:             ringBuf,
		AlarmSvc:            alarmSvc,
		SubRateLimitEvery:   250 * time.Microsecond,
		SubRateLimitBurst:   20,
		QueryRateLimitEvery: 20 * time.Millisecond,
		QueryRateLimitBurst: 10,
	})

	// init app
	app.NewApp(&app.Cfg{
		Ctx:                      ctx,
		Buf:                      ringBuf,
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
