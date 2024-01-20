/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"time"
)

type Info struct {
	Uptime  string         `json:"uptime"`
	Machine *MachineInfo   `json:"machine"`
	Buffer  *BufferInfo    `json:"buffer"`
	Alarms  []*AlarmStatus `json:"alarms"`
}

type MachineInfo struct {
	NumCpu int `json:"numCpu"`
}

type BufferInfo struct {
	Writes uint64 `json:"writes"`
	Size   uint32 `json:"size"`
}

type AlarmStatus struct {
	Name              string `json:"name"`
	Period            string `json:"period"`
	Threshold         int    `json:"threshold"`
	LenEvents         int    `json:"lenEvents"`
	TimeLastTriggered int64  `json:"timeLastTriggered"`
}

func (svc *HttpSvc) handleInfo(w http.ResponseWriter, r *http.Request) {
	if !svc.isAuthedForReading(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	data, err := json.Marshal(svc.info)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal info: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(data)
}

func (svc *HttpSvc) measureInfo() {
	numCpu := runtime.NumCPU()
	bufSize := svc.buf.Size()
	for {
		svc.info = &Info{
			Uptime: time.Since(svc.started).String(),
			Machine: &MachineInfo{
				NumCpu: numCpu,
			},
			Buffer: &BufferInfo{
				Writes: svc.buf.Writes.Load(),
				Size:   bufSize,
			},
			Alarms: make([]*AlarmStatus, 0, len(svc.alarmSvc.Alarms)),
		}
		for _, a := range svc.alarmSvc.Alarms {
			svc.info.Alarms = append(svc.info.Alarms, &AlarmStatus{
				Name:              a.Name,
				Period:            a.Period.String(),
				Threshold:         a.Threshold,
				LenEvents:         len(a.Events),
				TimeLastTriggered: a.LastTriggered.UnixMilli(),
			})
		}
		sort.Slice(svc.info.Alarms, func(i, j int) bool {
			return svc.info.Alarms[i].Name < svc.info.Alarms[j].Name
		})
		time.Sleep(time.Millisecond * 500)
	}
}
