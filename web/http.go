package web

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/swissinfo-ch/logd/ring"
)

type Webserver struct {
	ReadSecret string
	Buf        *ring.RingBuffer
	Started    time.Time
}

type Info struct {
	Writes  uint64 `json:"writes"`
	Started int64  `json:"started"`
}

func (s *Webserver) ServeHttp(laddr string) {
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
