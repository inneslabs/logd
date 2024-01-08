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
	readSecret     = []byte(os.Getenv("READ_SECRET"))
	writeSecret    = []byte(os.Getenv("WRITE_SECRET"))
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

	t := transport.NewTransporter(readSecret, writeSecret)
	go t.Listen(ctx, udpLaddr)

	a := alarm.NewSvc()
	a.Set(prodWpErrors())

	h := &web.Webserver{
		ReadSecret:  string(readSecret),
		Buf:         buf,
		Transporter: t,
		AlarmSvc:    a,
		Started:     time.Now(),
	}
	go h.ServeHttp(httpLaddr)

	go tailLogd(t, tailHost, tailReadSecret)

	go io(t, a)

	<-ctx.Done()
	fmt.Println("all routines ended")
}

func io(t *transport.Transporter, a *alarm.Svc) {
	for msg := range t.In {
		// pipe to tails
		t.Out <- msg
		// pipe to alarm svc
		a.In <- msg
		// write to buffer
		buf.Write(msg)
	}
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
