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
	slackWebhook := os.Getenv("LOGD_SLACK_WEBHOOK")

	// defaults
	appPort := 6101
	udpPort := 6102
	bufferSize := 1000000

	appPortEnv, set := os.LookupEnv("LOGD_APP_PORT")
	if set {
		var err error
		appPort, err = strconv.Atoi(appPortEnv)
		if err != nil {
			panic("LOGD_APP_PORT must be an int")
		}
	}

	udpPortEnv, set := os.LookupEnv("LOGD_UDP_PORT")
	if set {
		var err error
		udpPort, err = strconv.Atoi(udpPortEnv)
		if err != nil {
			panic("LOGD_UDP_PORT must be an int")
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

	// init ring buffer
	ringBuf := ring.NewRingBuffer(uint32(bufferSize))
	fmt.Printf("created ring buffer with capacity %d\n", bufferSize)

	// init alarms
	alarmSvc := alarm.NewSvc()
	alarmSvc.Set(prodWpErrors(slackWebhook))
	alarmSvc.Set(prodErrors(slackWebhook))

	// init root context
	ctx := getCtx()

	// init udp
	udp.NewSvc(&udp.Cfg{
		Ctx:                 ctx,
		LaddrPort:           udpPort,
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
		Ctx:            ctx,
		Buf:            ringBuf,
		AlarmSvc:       alarmSvc,
		RateLimitEvery: time.Second,
		RateLimitBurst: 100,
		Port:           appPort,
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
