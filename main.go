/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package main

import (
	"bufio"
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

	// init ring buffer
	bufferSize, err := strconv.ParseUint(bufferSizeStr, 10, 32)
	if err != nil {
		panic("BUFFER_SIZE must be an integer")
	}
	buf = ring.NewRingBuffer(uint32(bufferSize))
	fmt.Println("initialised buffer of size", bufferSize)

	// init root context
	ctx := getCtx()

	readFromStdin(ctx, buf)

	// init alarm svc
	alarmSvc := alarm.NewSvc()
	alarmSvc.Set(prodWpErrors(slackWebhook))
	alarmSvc.Set(prodErrors(slackWebhook))

	// init udp listener
	t := transport.NewTransporter(&transport.TransporterConfig{
		ReadSecret:  readSecret,
		WriteSecret: writeSecret,
		Buf:         buf,
		AlarmSvc:    alarmSvc,
	})
	go t.Listen(ctx, udpLaddr)

	// init webserver
	h := &web.Webserver{
		ReadSecret:  string(readSecret),
		Buf:         buf,
		Transporter: t,
		AlarmSvc:    alarmSvc,
		Started:     time.Now(),
	}
	go h.ServeHttp(httpLaddr)

	// maybe tail other instance for test data
	go tailLogd(buf, t, tailHost, tailReadSecret)

	fmt.Println("all routines started")
	// wait for kill signal
	<-ctx.Done()
	fmt.Println("all routines ended")
}

// cancelOnKillSig cancels the context on os interrupt kill signal
func cancelOnKillSig(sigs chan os.Signal, cancel context.CancelFunc) {
	switch <-sigs {
	case syscall.SIGINT:
		fmt.Println("\r\nreceived SIGINT")
	case syscall.SIGTERM:
		fmt.Println("\r\nreceived SIGTERM")
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

func readFromStdin(ctx context.Context, buf *ring.RingBuffer) {
	i := make(chan []byte)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		msg, _ := reader.ReadBytes('\n')
		i <- msg
	}()
	select {
	case msg := <-i:
		buf.Write(msg)
	case <-time.After(time.Second):
		fmt.Println("Timeout! No input received.")
	case <-ctx.Done():
		return
	}
}
