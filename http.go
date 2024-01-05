package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/fxamacker/cbor/v2"
	"github.com/intob/logd/logdentry"
)

type Webserver struct{}

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
	if r.Header.Get("authorization") != string(readSecret) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if r.URL.Path == "/" {
		s.handleRead(w, r)
		return
	}
	if r.URL.Path == "/info" {
		s.handleInfo(w, r)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func (s *Webserver) handleRead(w http.ResponseWriter, r *http.Request) {
	offset := 0
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "limit must be an integer", http.StatusBadRequest)
			return
		}
	}
	limit := 1000
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "limit must be an integer", http.StatusBadRequest)
			return
		}
		if limit > 10000 {
			limit = 10000
		}
	}
	envQ := r.URL.Query().Get("env")
	svcQ := r.URL.Query().Get("svc")
	fnQ := r.URL.Query().Get("fn")
	results := make([]*logdentry.Entry, 0)
	pages := 0
	for len(results) < limit && pages*limit < buf.Size()/10 {
		items := buf.Read(offset, limit)
		pages++
		offset += limit
		e := &logdentry.Entry{}
		for _, i := range items {
			err := cbor.Unmarshal(*i, e)
			if err != nil {
				fmt.Println("failed to unmarshal entry:", err)
				continue
			}
			if envQ != "" && envQ != e.Env {
				continue
			}
			if svcQ != "" && svcQ != e.Svc {
				continue
			}
			if fnQ != "" && fnQ != e.Fn {
				continue
			}
			results = append(results, e)
		}
	}
	data, err := json.Marshal(results)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal data: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(data)
}

func (s *Webserver) handleInfo(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(&Info{
		Writes:  buf.Writes.Load(),
		Started: started.UnixMilli(),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal info: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(data)
}
