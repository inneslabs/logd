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

	"github.com/intob/jfmt"
	"github.com/swissinfo-ch/logd/alarm"
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
	Name              string        `json:"name"`
	Period            string        `json:"period"`
	Threshold         int32         `json:"threshold"`
	EventCount        int32         `json:"eventCount"`
	TimeLastTriggered int64         `json:"timeLastTriggered"`
	Report            *alarm.Report `json:"report"`
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
			currentWrites := app.buf.NumWrites()
			delta := currentWrites - lastWrites
			timeDelta := time.Since(lastTime).Seconds()
			writePerSec := uint64(float64(delta) / timeDelta)
			if writePerSec > maxWritePerSec {
				maxWritePerSec = writePerSec
			}
			lastWrites = currentWrites
			lastTime = time.Now()

			alarms := make([]*alarm.Alarm, 0, 10)
			app.alarmSvc.Alarms.Range(func(key, value any) bool {
				al, ok := value.(*alarm.Alarm)
				if !ok {
					return true
				}
				alarms = append(alarms, al)
				return true
			})

			info := &Status{
				Commit: app.commit,
				Uptime: jfmt.FmtDuration(time.Since(app.started)),
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
					Period:            jfmt.FmtDuration(a.Period),
					Threshold:         a.Threshold,
					EventCount:        a.EventCount.Load(),
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
			return
		}
	}
}
