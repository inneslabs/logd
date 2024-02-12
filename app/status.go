/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"time"
)

type Status struct {
	Commit  string         `json:"commit"`
	Uptime  string         `json:"uptime"`
	Machine *MachineInfo   `json:"machine"`
	Buffer  *BufferInfo    `json:"buffer"`
	Alarms  []*AlarmStatus `json:"alarms"`
}

type MachineInfo struct {
	NumCpu int `json:"numCpu"`
}

type BufferInfo struct {
	Writes         uint64 `json:"writes"`
	Size           uint32 `json:"size"`
	MaxWritePerSec uint64 `json:"maxWritePerSec"`
}

type AlarmStatus struct {
	Name              string `json:"name"`
	Period            string `json:"period"`
	Threshold         int    `json:"threshold"`
	LenEvents         int    `json:"lenEvents"`
	TimeLastTriggered int64  `json:"timeLastTriggered"`
}

func (app *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(app.statusJson)))
	w.Write(app.statusJson)
}

func (app *App) measureStatus() {
	numCpu := runtime.NumCPU()
	bufSize := app.buf.Size()
	lastWrites := uint64(0)
	lastTime := time.Now()
	maxWritePerSec := uint64(0)

	for {
		select {
		case <-time.After(time.Second):
			currentWrites := app.buf.Writes.Load()
			delta := currentWrites - lastWrites
			timeDelta := time.Since(lastTime).Seconds()
			writePerSec := uint64(float64(delta) / timeDelta)
			if writePerSec > maxWritePerSec {
				maxWritePerSec = writePerSec
			}
			lastWrites = currentWrites
			lastTime = time.Now()

			alarms := app.alarmSvc.GetAll()

			info := &Status{
				Commit: app.commit,
				Uptime: time.Since(app.started).String(),
				Machine: &MachineInfo{
					NumCpu: numCpu,
				},
				Buffer: &BufferInfo{
					Writes:         currentWrites,
					Size:           bufSize,
					MaxWritePerSec: maxWritePerSec,
				},
				Alarms: make([]*AlarmStatus, 0, len(alarms)),
			}

			for _, a := range alarms {
				info.Alarms = append(info.Alarms, &AlarmStatus{
					Name:              a.Name,
					Period:            a.Period.String(),
					Threshold:         a.Threshold,
					LenEvents:         len(a.Events),
					TimeLastTriggered: a.LastTriggered.UnixMilli(),
				})
			}

			// Sort alarms by name
			sort.Slice(info.Alarms, func(i, j int) bool {
				return info.Alarms[i].Name < info.Alarms[j].Name
			})

			data, err := json.Marshal(info)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal json: %s", err))
			}

			app.statusJson = data
		case <-app.ctx.Done():
			fmt.Println("measureStatus ending")
		}
	}
}
