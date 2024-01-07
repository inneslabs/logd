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

const readRoutines = 10

var (
	buf         *ring.RingBuffer
	readSecret  []byte
	writeSecret []byte
)

func init() {
	bufferSize, err := strconv.ParseUint(os.Getenv("BUFFER_SIZE"), 10, 32)
	if err != nil {
		panic("BUFFER_SIZE must be an integer")
	}
	buf = ring.NewRingBuffer(uint32(bufferSize))
	fmt.Println("initialised buffer of size", bufferSize)
	readSecret = []byte(os.Getenv("READ_SECRET"))
	writeSecret = []byte(os.Getenv("WRITE_SECRET"))
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go cancelOnSig(sigs, cancel)

	h := &web.Webserver{
		ReadSecret: string(readSecret),
		Buf:        buf,
		Started:    time.Now(),
	}
	go h.ServeHttp(os.Getenv("HTTP_LADDR"))

	t := transport.NewTransporter()
	t.SetReadSecret(readSecret)
	t.SetWriteSecret(writeSecret)
	go t.Listen(ctx, os.Getenv("UDP_LADDR"))

	a := alarm.NewSvc()
	a.Set(prodWpErrors())
	a.Set(prodWarnings())

	for i := 0; i < readRoutines; i++ {
		go readIn(ctx, t, a)
	}

	<-ctx.Done()
	fmt.Println("all routines ended")
}

func readIn(ctx context.Context, t *transport.Transporter, a *alarm.Svc) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-t.In:
			t.Out <- msg
			buf.Write(msg)
			a.In <- msg
		}
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
