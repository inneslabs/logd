package alarm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/swissinfo-ch/logd/cmd"
)

type Report struct {
	Svcs map[string]*SvcReport `json:"svcs"`
}

type SvcReport struct {
	Fns map[string]*FnReport `json:"fns"`
}

type FnReport struct {
	EventCount int32    `json:"eventCount"`
	Sample     *cmd.Msg `json:"sample"`
}

// helper to flatten the report
type svcFnPair struct {
	SvcName    string   `json:"svcName"`
	FnName     string   `json:"fnName"`
	EventCount int32    `json:"eventCount"`
	Sample     *cmd.Msg `json:"sample"`
}

// GenerateTopNView condenses some of the report into a string
func GenerateTopNView(report *Report, topN int) string {
	var pairs []svcFnPair

	// Flatten the structure
	for svcName, svcReport := range report.Svcs {
		for fnName, fnReport := range svcReport.Fns {
			pairs = append(pairs, svcFnPair{
				SvcName:    svcName,
				FnName:     fnName,
				EventCount: fnReport.EventCount,
				Sample:     fnReport.Sample,
			})
		}
	}

	// Sort by EventCount in descending order
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].EventCount > pairs[j].EventCount
	})

	// Limit to top N and format the output
	var reportLines []string
	for i, pair := range pairs {
		if i >= topN {
			break
		}
		reportLines = append(reportLines, fmt.Sprintf("%s/%s: %d sample: %s",
			pair.SvcName,
			pair.FnName,
			pair.EventCount,
			pair.Sample.GetTxt()))
	}

	return strings.Join(reportLines, "\n")
}

func (a *Alarm) createReport() {
	r := &Report{
		Svcs: make(map[string]*SvcReport),
	}
	a.Events.Range(func(k, v interface{}) bool {
		e := v.(*Event)
		svc := e.Msg.GetSvc()
		fn := e.Msg.GetFn()
		if _, ok := r.Svcs[svc]; !ok {
			r.Svcs[svc] = &SvcReport{
				Fns: make(map[string]*FnReport),
			}
		}
		if _, ok := r.Svcs[svc].Fns[fn]; !ok {
			r.Svcs[svc].Fns[fn] = &FnReport{}
		}
		r.Svcs[svc].Fns[fn].EventCount++
		// TODO maybe collect more than one sample
		r.Svcs[svc].Fns[fn].Sample = e.Msg
		return true
	})
	a.Report = r
}
