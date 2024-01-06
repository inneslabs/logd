package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
)

func cancelOnSig(sigs chan os.Signal, cancel context.CancelFunc) {
	switch <-sigs {
	case syscall.SIGINT:
		fmt.Println("\r\nreceived SIGINT")
	case syscall.SIGTERM:
		fmt.Println("\r\nreceived SIGTERM")
	}
	cancel()
}
