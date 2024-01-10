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

var (
	buf            *ring.RingBuffer
	bufferSizeStr  = os.Getenv("BUFFER_SIZE")
	httpLaddr      = os.Getenv("HTTP_LADDR")
	udpLaddr       = os.Getenv("UDP_LADDR")
	readSecret     = os.Getenv("READ_SECRET")
	writeSecret    = os.Getenv("WRITE_SECRET")
	slackWebhook   = os.Getenv("SLACK_WEBHOOK")
	tailHost       = os.Getenv("TAIL_HOST")
	tailReadSecret = os.Getenv("TAIL_READ_SECRET")
)

func init() {
	bufferSize, err := strconv.ParseUint(bufferSizeStr, 10, 32)
	if err != nil {
		panic("BUFFER_SIZE must be an integer")
	}
	buf = ring.NewRingBuffer(uint32(bufferSize))
	fmt.Println("initialised buffer of size", bufferSize)
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go cancelOnSig(sigs, cancel)

	alarmSvc := alarm.NewSvc()
	alarmSvc.Set(prodWpErrors())
	alarmSvc.Set(prodErrors())

	t := transport.NewTransporter(&transport.TransporterConfig{
		ReadSecret:  readSecret,
		WriteSecret: writeSecret,
		Buf:         buf,
		AlarmSvc:    alarmSvc,
	})
	go t.Listen(ctx, udpLaddr)

	h := &web.Webserver{
		ReadSecret:  string(readSecret),
		Buf:         buf,
		Transporter: t,
		AlarmSvc:    alarmSvc,
		Started:     time.Now(),
	}
	go h.ServeHttp(httpLaddr)

	go tailLogd(t, tailHost, tailReadSecret)

	<-ctx.Done()
	fmt.Println("all routines ended")
}

func cancelOnSig(sigs chan os.Signal, cancel context.CancelFunc) {
	switch <-sigs {
	case syscall.SIGINT:
		fmt.Println("\r\nreceived SIGINT")
	case syscall.SIGTERM:
		fmt.Println("\r\nreceived SIGTERM")
	}
	cancel()
}
