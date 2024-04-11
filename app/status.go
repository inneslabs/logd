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
)

type Status struct {
	Commit   string     `json:"commit"`
	Uptime   string     `json:"uptime"`
	NCpu     int        `json:"ncpu"`
	MemAlloc uint64     `json:"mem_alloc"`
	MemSys   uint64     `json:"mem_sys"`
	Store    *StoreInfo `json:"store"`
}

type StoreInfo struct {
	NWrites uint64      `json:"nwrites"`
	MaxRate uint64      `json:"max_rate"`
	Rings   []*RingInfo `json:"rings"`
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
	ncpu := runtime.NumCPU()
	lastWrites := uint64(0)
	lastTime := time.Now()
	maxRate := uint64(0)
	for {
		<-time.After(time.Second)
		writes := app.logStore.NWrites()
		delta := writes - lastWrites
		timeDelta := time.Since(lastTime).Seconds()
		rate := uint64(float64(delta) / timeDelta)
		if rate > maxRate {
			maxRate = rate
		}
		lastWrites = writes
		lastTime = time.Now()

		headsAndSizes := app.logStore.HeadsAndSizes()
		rings := make([]*RingInfo, 0, len(headsAndSizes))
		for key := range headsAndSizes {
			rings = append(rings, &RingInfo{
				Key:  key,
				Head: headsAndSizes[key][0],
				Size: headsAndSizes[key][1],
			})
		}
		sort.Slice(rings, func(i, j int) bool {
			return rings[i].Key < rings[j].Key
		})

		memStats := &runtime.MemStats{}
		runtime.ReadMemStats(memStats)

		info := &Status{
			Commit:   app.commit,
			Uptime:   jfmt.FmtDuration(time.Since(app.started)),
			NCpu:     ncpu,
			MemAlloc: memStats.HeapAlloc,
			MemSys:   memStats.Sys,
			Store: &StoreInfo{
				NWrites: writes,
				Rings:   rings,
				MaxRate: maxRate,
			},
		}

		data, err := json.Marshal(info)
		if err != nil {
			panic(fmt.Sprintf("failed to marshal json: %s", err))
		}

		app.statusJson = data

	}
}
