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

	"github.com/intob/jfmt"
	"github.com/swissinfo-ch/logd/app"
	"github.com/swissinfo-ch/logd/store"
	"github.com/swissinfo-ch/logd/udp"
)

func main() {
	// no default, can also be blank
	readSecret := os.Getenv("LOGD_READ_SECRET")
	writeSecret := os.Getenv("LOGD_WRITE_SECRET")
	//slackWebhook := os.Getenv("LOGD_SLACK_WEBHOOK")

	// defaults
	udpLaddrPort := ":6102" // string supports fly-global-services:6102
	appPort := 6101
	accessControlAllowOrigin := "*"

	udpPortEnv, set := os.LookupEnv("LOGD_UDP_LADDRPORT")
	if set {
		udpLaddrPort = udpPortEnv
	}

	appPortEnv, set := os.LookupEnv("LOGD_APP_PORT")
	if set {
		var err error
		appPort, err = strconv.Atoi(appPortEnv)
		if err != nil {
			panic("LOGD_APP_PORT must be an int")
		}
	}

	accessControlAllowOriginEnv, set := os.LookupEnv("LOGD_ACCESS_CONTROL_ALLOW_ORIGIN")
	if set {
		accessControlAllowOrigin = accessControlAllowOriginEnv
	}

	// init store
	svcSize := uint32(10000) // 10K logs per env/svc
	logStore := store.NewStore(&store.Cfg{
		RingSizes: map[string]uint32{
			"/prod/wp":              500000,
			"/prod/ticker-service":      svcSize,
			"/prod/taxonomy-service":    svcSize,
			"/prod/swiplus-service":     svcSize,
			"/prod/video-service":       svcSize,
			"/prod/rss-service":         svcSize,
			"/prod/swiplus-app-backend": svcSize,
			"/prod/newsletter-service":  svcSize,
			"/prod/archive-service":     svcSize,
			"/prod/swi-core":            1000,
		},
		FallbackSize: 500000,
	})

	// print config insensitive config
	fmt.Println("udp port set to", udpLaddrPort)
	fmt.Println("app port set to", appPort)
	fmt.Println("access-control-allow-origin set to", accessControlAllowOrigin)
	for key, size := range logStore.Sizes() {
		fmt.Printf("%s:%s\n", jfmt.FmtCount32(size), key)
	}

	// init root context
	ctx := getCtx()

	// init udp
	udp.NewSvc(&udp.Cfg{
		Ctx:                 ctx,
		LaddrPort:           udpLaddrPort,
		ReadSecret:          readSecret,
		WriteSecret:         writeSecret,
		LogStore:            logStore,
		AlarmSvc:            nil,
		SubRateLimitEvery:   100 * time.Microsecond,
		SubRateLimitBurst:   50,
		QueryRateLimitEvery: 20 * time.Millisecond,
		QueryRateLimitBurst: 10,
	})

	// init app
	app.NewApp(&app.Cfg{
		Ctx:                      ctx,
		LogStore:                 logStore,
		AlarmSvc:                 nil,
		RateLimitEvery:           time.Second,
		RateLimitBurst:           10,
		Port:                     appPort,
		AccessControlAllowOrigin: accessControlAllowOrigin,
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
