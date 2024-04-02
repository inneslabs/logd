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
	"gopkg.in/yaml.v3"
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
	fmt.Println("🌱 running", string(commit))
	config := &Cfg{
		Udp: &udp.Cfg{
			Ctx:            ctx,
			WorkerPoolSize: runtime.NumCPU(),
			LaddrPort:      ":6102",
			Secrets: &udp.Secrets{
				Read:  "gold",
				Write: "bitcoin",
			},
			Guard: &guard.Cfg{
				HistorySize: 1000,
				SumTtl:      100 * time.Millisecond,
			},
			TailRateLimitEvery:  50 * time.Microsecond,
			TailRateLimitBurst:  4,
			QueryRateLimitEvery: 50 * time.Microsecond,
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
	err = cfg.Load("logdrc.yml", "/etc", config)
	if err != nil {
		fmt.Println("no config file loaded")
	}
	secYml, err := os.ReadFile("secrets.yml")
	if err == nil {
		sec := &udp.Secrets{}
		err = yaml.Unmarshal(secYml, sec)
		if err != nil {
			panic(err)
		} else {
			config.Udp.Secrets = sec
			fmt.Println("secrets loaded from secrets.yml")
		}
	}
	logStore := store.NewStore(config.Store)
	config.App.LogStore = logStore
	config.Udp.LogStore = logStore
	udp.NewSvc(config.Udp)
	app.NewApp(config.App)
	fmt.Println("read secret sha256:", secretHash(config.Udp.Secrets.Read))
	fmt.Println("write secret sha256:", secretHash(config.Udp.Secrets.Write))
	fmt.Printf("guard: %+v\n", config.Udp.Guard)
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
