/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package web

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Info struct {
	Uptime  string         `json:"uptime"`
	Machine *MachineInfo   `json:"machine"`
	Buffer  *BufferInfo    `json:"buffer"`
	Alarms  []*AlarmStatus `json:"alarms"`
}

type MachineInfo struct {
	MemTotalMB float64 `json:"memTotalMB"`
	MemAllocMB float64 `json:"memAllocMB"`
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

var (
	info        *Info
	totalMemory float64
)

func init() {
	var err error
	totalMemory, err = readTotalMemory()
	if err != nil {
		fmt.Println("failed to read total memory:", err)
	}
}

func (s *Webserver) handleInfo(w http.ResponseWriter, r *http.Request) {
	if !s.isAuthedForReading(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	data, err := json.Marshal(info)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal info: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(data)
}

func (s *Webserver) measureInfo() {
	for {
		memStats := &runtime.MemStats{}
		runtime.ReadMemStats(memStats)
		info = &Info{
			Uptime: time.Since(s.Started).String(),
			Machine: &MachineInfo{
				MemAllocMB: float64(memStats.Alloc) / 1024 / 1024,
				MemTotalMB: totalMemory,
			},
			Buffer: &BufferInfo{
				Writes: s.Buf.Writes.Load(),
				Size:   s.Buf.Size(),
			},
			Alarms: make([]*AlarmStatus, 0),
		}
		for _, a := range s.AlarmSvc.Alarms {
			info.Alarms = append(info.Alarms, &AlarmStatus{
				Name:              a.Name,
				Period:            a.Period.String(),
				Threshold:         a.Threshold,
				LenEvents:         len(a.Events),
				TimeLastTriggered: a.LastTriggered.UnixMilli(),
			})
		}
		sort.Slice(info.Alarms, func(i, j int) bool {
			return info.Alarms[i].Name < info.Alarms[j].Name
		})
		time.Sleep(time.Millisecond * 500)
	}
}

func readTotalMemory() (float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("error opening meminfo: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0, fmt.Errorf("unexpected format in meminfo")
			}
			memTotalKB, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("error parsing memory value: %w", err)
			}
			return float64(memTotalKB) / 1024, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading meminfo: %w", err)
	}
	return 0, errors.New("MemTotal not found in file")
}
