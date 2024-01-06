package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/swissinfo-ch/logd/ring"
	"github.com/swissinfo-ch/logd/transport"
)

var (
	buf         *ring.RingBuffer
	readSecret  []byte
	writeSecret []byte
	started     time.Time
)

func init() {
	bufferSize, err := strconv.Atoi(os.Getenv("BUFFER_SIZE"))
	if err != nil {
		panic("BUFFER_SIZE must be an integer")
	}
	buf = ring.NewRingBuffer(bufferSize)
	readSecret = []byte(os.Getenv("READ_SECRET"))
	writeSecret = []byte(os.Getenv("WRITE_SECRET"))
	started = time.Now()
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go cancelOnSig(sigs, cancel)

	t := transport.NewTransporter()
	t.SetReadSecret(readSecret)
	t.SetWriteSecret(writeSecret)
	go readIn(ctx, t)
	go t.Listen(ctx, os.Getenv("UDP_LADDR"))

	h := &Webserver{}
	go h.ServeHttp(os.Getenv("HTTP_LADDR"))

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

func readIn(ctx context.Context, t *transport.Transporter) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-t.In:
			t.Out <- msg
			buf.Write(msg)
		}
	}
}
