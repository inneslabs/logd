package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/inneslabs/jfmt"
)

type Status struct {
	Commit  string       `json:"commit"`
	Uptime  string       `json:"uptime"`
	Machine *MachineInfo `json:"machine"`
	Store   *StoreInfo   `json:"store"`
}

type MachineInfo struct {
	NumCpu int `json:"numCpu"`
}

type StoreInfo struct {
	Writes uint64 `json:"writes"`
	// iterables are easier than maps in JS
	Rings          []*RingInfo `json:"rings"`
	MaxWritePerSec uint64      `json:"maxWritePerSec"`
}

type RingInfo struct {
	Key  string `json:"key"`
	Head uint32 `json:"head"`
	Size uint32 `json:"size"`
}

func (app *App) handleStatus(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(app.statusJson)))
	w.Write(app.statusJson)
}

func (app *App) measureStatus() {
	numCpu := runtime.NumCPU()
	lastWrites := uint64(0)
	lastTime := time.Now()
	maxWritePerSec := uint64(0)

	for {
		select {
		case <-time.After(time.Second):
			writes := app.logStore.NumWrites()
			delta := writes - lastWrites
			timeDelta := time.Since(lastTime).Seconds()
			writePerSec := uint64(float64(delta) / timeDelta)
			if writePerSec > maxWritePerSec {
				maxWritePerSec = writePerSec
			}
			lastWrites = writes
			lastTime = time.Now()

			// build log store rings report
			heads := app.logStore.Heads()
			sizes := app.logStore.Sizes()
			rings := make([]*RingInfo, 0, len(heads))
			for key := range heads {
				rings = append(rings, &RingInfo{
					Key:  key,
					Head: heads[key],
					Size: sizes[key],
				})
			}
			sort.Slice(rings, func(i, j int) bool {
				return rings[i].Key < rings[j].Key
			})

			info := &Status{
				Commit: app.commit,
				Uptime: jfmt.FmtDuration(time.Since(app.started)),
				Machine: &MachineInfo{
					NumCpu: numCpu,
				},
				Store: &StoreInfo{
					Writes:         writes,
					Rings:          rings,
					MaxWritePerSec: maxWritePerSec,
				},
			}

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
