package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/intob/logd/app"
	"github.com/intob/logd/guard"
	"github.com/intob/logd/store"
	"github.com/intob/logd/udp"
	"gopkg.in/yaml.v3"
)

type Cfg struct {
	Udp   *udp.Cfg   `yaml:"udp"`
	App   *app.Cfg   `yaml:"app"`
	Store *store.Cfg `yaml:"store"`
}

func main() {
	const (
		secretsFile = "/etc/intob/logd/secrets.yml"
		configFile  = "/etc/intob/logd/config.yml"
	)
	ctx := rootCtx()
	commit, err := os.ReadFile("/etc/logd/commit")
	if err != nil {
		fmt.Println("failed to read commit file:", err)
	}
	fmt.Println("🌱 running", string(commit))
	config := &Cfg{
		Udp: &udp.Cfg{
			LaddrPort:        ":6102",
			PacketBufferSize: 1460,  // typical WiFi MTU, feel free to increase
			QueryHardLimit:   50000, // maybe use offset
			Secrets: &udp.Secrets{
				Read:  "gold",
				Write: "bitcoin",
			},
			Guard: &guard.Cfg{
				FilterCap: 16000000, // balance for memory, larger size reduces false positives
				// now I know, we need 2 smaller filters, A & B.
				// Each must pass, and each is reset off-kilter to prevent
				// replay immediately after filter reset.
				FilterTtl: 10 * time.Second,
				PacketTtl: 200 * time.Millisecond, // keep to a minimum, reduce reliance on filter
			},
		},
		App: &app.Cfg{
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
	err = loadYml(configFile, config)
	if err != nil {
		fmt.Printf("err loading %q: %v\n", configFile, err)
	}
	sec := &udp.Secrets{}
	err = loadYml(secretsFile, sec)
	if err != nil {
		fmt.Printf("err loading %q: %v\n", secretsFile, err)
	} else {
		config.Udp.Secrets = sec
		fmt.Printf("secrets loaded from %q\n", secretsFile)
	}
	logStore := store.NewStore(config.Store)
	config.App.LogStore = logStore
	config.Udp.LogStore = logStore
	udp.NewSvc(ctx, config.Udp)
	app.NewApp(ctx, config.App)
	fmt.Println("read secret sha256:", secretHash(config.Udp.Secrets.Read))
	fmt.Println("write secret sha256:", secretHash(config.Udp.Secrets.Write))
	fmt.Printf("udp: %+v\n", config.Udp)
	fmt.Printf("udp guard: %+v\n", config.Udp.Guard)
	<-ctx.Done()
	<-time.After(time.Millisecond)
	fmt.Println("logd ended")
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

func loadYml(fname string, v interface{}) error {
	file, err := os.OpenFile(fname, os.O_RDONLY, 0o777)
	if err != nil {
		return fmt.Errorf("err opening file: %w", err)
	}
	defer file.Close()
	dec := yaml.NewDecoder(file)
	err = dec.Decode(v)
	if err != nil {
		return fmt.Errorf("err decoding cfg file (%s): %w", fname, err)
	}
	return nil
}
