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
	"github.com/swissinfo-ch/logd/transport"
	"github.com/swissinfo-ch/logd/web"
)

func main() {
	var (
		buf           *ring.RingBuffer
		bufferSizeStr = os.Getenv("BUFFER_SIZE")
		httpLaddrPort = os.Getenv("HTTP_LADDRPORT")
		udpLaddrPort  = os.Getenv("UDP_LADDRPORT")
		readSecret    = os.Getenv("READ_SECRET")
		writeSecret   = os.Getenv("WRITE_SECRET")
		slackWebhook  = os.Getenv("SLACK_WEBHOOK")
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
	buf = ring.NewRingBuffer(uint32(bufferSize))
	fmt.Printf("created ring buffer with %d slots\n", bufferSize)

	// init alarm svc
	alarmSvc := alarm.NewSvc()
	alarmSvc.Set(prodWpErrors(slackWebhook))
	alarmSvc.Set(prodErrors(slackWebhook))

	// init root context
	ctx := getCtx()

	// init udp listener
	t := transport.NewTransporter(&transport.Config{
		ReadSecret:  readSecret,
		WriteSecret: writeSecret,
		Buf:         buf,
		AlarmSvc:    alarmSvc,
	})
	go t.Listen(ctx)

	// init webserver
	h := &web.Webserver{
		ReadSecret:  string(readSecret),
		Buf:         buf,
		Transporter: t,
		AlarmSvc:    alarmSvc,
		Started:     time.Now(),
	}
	go h.ServeHttp(httpLaddrPort)

	fmt.Println("all routines started")
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
