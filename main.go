package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/inneslabs/cfg"
	"github.com/inneslabs/logd/app"
	"github.com/inneslabs/logd/guard"
	"github.com/inneslabs/logd/store"
	"github.com/inneslabs/logd/udp"
)

type Cfg struct {
	Udp   *udp.Cfg   `yaml:"udp"`
	App   *app.Cfg   `yaml:"app"`
	Store *store.Cfg `yaml:"store"`
}

func main() {
	ctx := rootCtx()
	commit, err := os.ReadFile("commit")
	if err != nil {
		fmt.Println("failed to read commit file:", err)
	}
	config := &Cfg{
		Udp: &udp.Cfg{
			Ctx:            ctx,
			WorkerPoolSize: runtime.NumCPU(),
			LaddrPort:      ":6102",
			ReadSecret:     "gold",
			WriteSecret:    "bitcoin",
			Guard: &guard.Cfg{
				HistorySize: 1000,
				SumTtl:      100 * time.Millisecond,
			},
			SubRateLimitEvery:   100 * time.Microsecond,
			SubRateLimitBurst:   4,
			QueryRateLimitEvery: 100 * time.Microsecond,
			QueryRateLimitBurst: 4,
		},
		App: &app.Cfg{
			Ctx:                      ctx,
			Commit:                   commit,
			LaddrPort:                ":6101",
			RateLimitEvery:           200 * time.Millisecond,
			RateLimitBurst:           10,
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
		panic(err)
	}
	readSecretEnv, set := os.LookupEnv("LOGD_READ_SECRET")
	if set {
		config.Udp.ReadSecret = readSecretEnv
		fmt.Println("read secret loaded from env var")
	}
	writeSecretEnv, set := os.LookupEnv("LOGD_WRITE_SECRET")
	if set {
		config.Udp.WriteSecret = writeSecretEnv
		fmt.Println("write secret loaded from env var")
	}
	logStore := store.NewStore(config.Store)
	config.App.LogStore = logStore
	config.Udp.LogStore = logStore
	udp.NewSvc(config.Udp)
	app.NewApp(config.App)
	fmt.Println("🌱 running", string(commit))
	fmt.Println("guard cfg:", config.Udp.Guard)
	fmt.Println("read secret sha256:", secretHash(config.Udp.ReadSecret))
	fmt.Println("write secret sha256:", secretHash(config.Udp.WriteSecret))
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

// rootCtx returns a root context that awaits a kill signal from os
func rootCtx() context.Context {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go cancelOnKillSig(sigs, cancel)
	return ctx
}

func secretHash(secret string) string {
	readSecretSum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(readSecretSum[:])
}
