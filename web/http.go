/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package web

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/ring"
	"github.com/swissinfo-ch/logd/udp"
	"golang.org/x/time/rate"
)

type HttpSvc struct {
	readSecret  string
	buf         *ring.RingBuffer
	udpSvc      *udp.UdpSvc
	alarmSvc    *alarm.Svc
	started     time.Time
	rateLimiter *rate.Limiter
	info        *Info
}

type Config struct {
	ReadSecret     string
	Buf            *ring.RingBuffer
	UdpSvc         *udp.UdpSvc
	AlarmSvc       *alarm.Svc
	RateLimitEvery time.Duration
	RateLimitBurst int
}

func NewHttpSvc(cfg *Config) *HttpSvc {
	return &HttpSvc{
		readSecret:  cfg.ReadSecret,
		buf:         cfg.Buf,
		udpSvc:      cfg.UdpSvc,
		alarmSvc:    cfg.AlarmSvc,
		started:     time.Now(),
		rateLimiter: rate.NewLimiter(rate.Every(cfg.RateLimitEvery), cfg.RateLimitBurst),
	}
}

func (svc *HttpSvc) ServeHttp(laddrPort string) {
	go svc.measureInfo()
	http.Handle("/", http.HandlerFunc(svc.handleRequest))
	fmt.Println("listening http on " + laddrPort)
	err := http.ListenAndServe(laddrPort, nil)
	if err != nil {
		fmt.Println("failed to start http server: " + err.Error())
		os.Exit(1)
	}
}

func (svc *HttpSvc) handleRequest(w http.ResponseWriter, r *http.Request) {
	svc.rateLimiter.Wait(r.Context())
	w.Header().Set("access-control-allow-origin", "*")
	w.Header().Set("access-control-allow-methods", "GET, POST, OPTIONS")
	w.Header().Set("access-control-allow-headers", "authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.URL.Path == "/" {
		svc.handleQuery(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/info") {
		svc.handleInfo(w, r)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func (svc *HttpSvc) isAuthedForReading(r *http.Request) bool {
	return r.Header.Get("authorization") == svc.readSecret
}
