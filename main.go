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

const readRoutines = 10

var (
	buf                *ring.RingBuffer
	readSecret         []byte
	writeSecret        []byte
	started            time.Time
	willTailForTesting bool
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
	started = time.Now()
	willTailForTesting = os.Getenv("TAIL_FOR_TESTING") != ""
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go cancelOnSig(sigs, cancel)

	t := transport.NewTransporter()
	t.SetReadSecret(readSecret)
	t.SetWriteSecret(writeSecret)
	for i := 0; i < readRoutines; i++ {
		go readIn(ctx, t)
	}
	go t.Listen(ctx, os.Getenv("UDP_LADDR"))

	h := &Webserver{}
	go h.ServeHttp(os.Getenv("HTTP_LADDR"))

	<-ctx.Done()
	fmt.Println("all routines ended")
}

func readIn(ctx context.Context, t *transport.Transporter) {
	if willTailForTesting {
		go tailForTesting(t)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-t.In:
			t.Out <- msg
			buf.Write(msg)
			// alarmSvc.In <- msg
		}
	}
}
