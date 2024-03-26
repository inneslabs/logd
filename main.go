package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/inneslabs/logd/app"
	"github.com/inneslabs/logd/cfg"
	"github.com/inneslabs/logd/store"
	"github.com/inneslabs/logd/udp"
)

func main() {
	// defaults
	config := &cfg.LogdCfg{
		UdpLaddrPort: ":6102",
		AppSettings: &cfg.AppSettings{
			LaddrPort:                ":6101",
			AccessControlAllowOrigin: "*",
		},
		Store: &store.Cfg{
			FallbackSize: 100000,
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	err = cfg.Load("logdrc.yml", wd, config)
	if err != nil {
		fmt.Println(err)
	}

	// env vars overwrite config file
	readSecret, set := os.LookupEnv("LOGD_READ_SECRET")
	if set {
		config.ReadSecret = readSecret
		fmt.Println("read secret set from env LOGD_READ_SECRET")
	}
	writeSecret, set := os.LookupEnv("LOGD_WRITE_SECRET")
	if set {
		config.WriteSecret = writeSecret
		fmt.Println("write secret set from env LOGD_WRITE_SECRET")
	}

	logStore := store.NewStore(config.Store)

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
		SubRateLimitBurst:   20,
		QueryRateLimitEvery: 100 * time.Microsecond,
		QueryRateLimitBurst: 20,
	})

	// init app
	app.NewApp(&app.Cfg{
		Ctx:            ctx,
		Settings:       config.AppSettings,
		LogStore:       logStore,
		RateLimitEvery: 500 * time.Millisecond,
		RateLimitBurst: 10,
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
