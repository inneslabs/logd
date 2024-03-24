package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/inneslabs/jfmt"
	"github.com/inneslabs/logd/app"
	"github.com/inneslabs/logd/cfg"
	"github.com/inneslabs/logd/store"
	"github.com/inneslabs/logd/udp"
)

func main() {
	// defaults
	config := &cfg.LogdCfg{
		UdpLaddrPort:             ":6102",
		AppPort:                  6101,
		AccessControlAllowOrigin: "*",
		Store: &store.Cfg{
			FallbackSize: 100000,
		},
	}

	err := cfg.Load("logdrc.yml", config)
	if err != nil {
		fmt.Println(err)
	}

	logStore := store.NewStore(config.Store)

	// print config insensitive config
	fmt.Println("udp port set to", config.UdpLaddrPort)
	fmt.Println("app port set to", config.AppPort)
	fmt.Println("access-control-allow-origin set to", config.AccessControlAllowOrigin)
	for key, size := range logStore.Sizes() {
		fmt.Printf("%s: %s\n", key, jfmt.FmtCount32(size))
	}

	// init root context
	ctx := getCtx()

	// init udp
	udp.NewSvc(&udp.Cfg{
		Ctx:                 ctx,
		LaddrPort:           config.UdpLaddrPort,
		ReadSecret:          config.ReadSecret,
		WriteSecret:         config.WriteSecret,
		LogStore:            logStore,
		SubRateLimitEvery:   100 * time.Microsecond,
		SubRateLimitBurst:   50,
		QueryRateLimitEvery: 20 * time.Millisecond,
		QueryRateLimitBurst: 10,
	})

	// init app
	app.NewApp(&app.Cfg{
		Ctx:                      ctx,
		LogStore:                 logStore,
		RateLimitEvery:           time.Second,
		RateLimitBurst:           10,
		Port:                     config.AppPort,
		AccessControlAllowOrigin: config.AccessControlAllowOrigin,
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
