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
	"github.com/swissinfo-ch/logd/ring"
	"github.com/swissinfo-ch/logd/udp"
	"github.com/swissinfo-ch/logd/web"
)

func main() {
	var (
		bufferSizeStr = os.Getenv("LOGD_BUFFER_SIZE")
		httpLaddrPort = os.Getenv("LOGD_HTTP_LADDRPORT")
		udpLaddrPort  = os.Getenv("LOGD_UDP_LADDRPORT")
		readSecret    = os.Getenv("LOGD_READ_SECRET")
		writeSecret   = os.Getenv("LOGD_WRITE_SECRET")
		slackWebhook  = os.Getenv("LOGD_SLACK_WEBHOOK")
		ringBuf       *ring.RingBuffer
	)

	// defaults
	if httpLaddrPort == "" {
		httpLaddrPort = ":6101"
	}
	if udpLaddrPort == "" {
		udpLaddrPort = ":6102"
	}

	// init ring buffer
	bufferSize, err := strconv.ParseUint(bufferSizeStr, 10, 32)
	if err != nil {
		bufferSize = 1000000
	}
	ringBuf = ring.NewRingBuffer(uint32(bufferSize))
	fmt.Printf("created ring buffer with %d slots\n", bufferSize)

	// init alarm svc
	alarmSvc := alarm.NewSvc()
	alarmSvc.Set(prodWpErrors(slackWebhook))
	alarmSvc.Set(prodErrors(slackWebhook))

	// init root context
	ctx := getCtx()

	// init udp svc
	udpSvc := udp.NewSvc(&udp.Config{
		LaddrPort:           udpLaddrPort,
		ReadSecret:          readSecret,
		WriteSecret:         writeSecret,
		RingBuf:             ringBuf,
		AlarmSvc:            alarmSvc,
		SubRateLimitEvery:   100 * time.Microsecond,
		SubRateLimitBurst:   10,
		QueryRateLimitEvery: 10 * time.Millisecond,
		QueryRateLimitBurst: 10,
	})
	go udpSvc.Listen(ctx)

	// init http svc
	httpSvc := web.NewHttpSvc(&web.Config{
		ReadSecret:     readSecret,
		Buf:            ringBuf,
		UdpSvc:         udpSvc,
		AlarmSvc:       alarmSvc,
		RateLimitEvery: 250 * time.Millisecond,
		RateLimitBurst: 10,
	})
	go httpSvc.ServeHttp(httpLaddrPort)

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
