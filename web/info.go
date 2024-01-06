package web

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Webserver) handleInfo(w http.ResponseWriter, r *http.Request) {
	if !s.isAuthedForReading(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	data, err := json.Marshal(&Info{
		Writes:  s.Buf.Writes.Load(),
		Started: s.Started.UnixMilli(),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal info: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(data)
}
