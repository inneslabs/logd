package web

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/swissinfo-ch/logd/alarm"
	"github.com/swissinfo-ch/logd/ring"
	"github.com/swissinfo-ch/logd/transport"
)

type Webserver struct {
	ReadSecret  string
	Buf         *ring.RingBuffer
	Transporter *transport.Transporter
	AlarmSvc    *alarm.Svc
	Started     time.Time
}

func (s *Webserver) ServeHttp(laddr string) {
	go s.measureInfo()
	http.Handle("/", http.HandlerFunc(s.handleRequest))
	fmt.Println("listening http on " + laddr)
	err := http.ListenAndServe(laddr, nil)
	if err != nil {
		fmt.Println("failed to start http server: " + err.Error())
		os.Exit(1)
	}
}

func (s *Webserver) handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("access-control-allow-origin", "*")
	w.Header().Set("access-control-allow-methods", "GET, POST, OPTIONS")
	w.Header().Set("access-control-allow-headers", "authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.URL.Path == "/" {
		s.handleRead(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/info") {
		s.handleInfo(w, r)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func (s *Webserver) isAuthedForReading(r *http.Request) bool {
	return r.Header.Get("authorization") == s.ReadSecret
}
