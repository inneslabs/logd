/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package alarm

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/swissinfo-ch/logd/cmd"
	"github.com/swissinfo-ch/logd/ring"
)

type AlarmSvc struct {
	in        chan *cmd.Msg // buffer doesn't help
	triggered chan *Alarm   // buffer doesn't help
	Alarms    sync.Map
	started   time.Time
}

type Alarm struct {
	Name      string
	Recent    *ring.Ring
	Period    time.Duration // period of analysis
	Threshold int32
	// using sync.Map for better concurrent access
	Events        sync.Map // key is unix milli
	EventCount    atomic.Int32
	Action        func() error
	LastTriggered time.Time
}

type Event struct {
	Msg      *cmd.Msg
	Occurred time.Time
}

func NewSvc() *AlarmSvc {
	s := &AlarmSvc{
		in:        make(chan *cmd.Msg, 100),
		triggered: make(chan *Alarm),
		started:   time.Now(),
	}
	// we need some gophers
	go s.matchMsgs()
	go s.kickOldMatchedEvents()
	go s.callActions()
	return s
}

func (svc *AlarmSvc) Set(al *Alarm) {
	svc.Alarms.Store(al.Name, al)
	fmt.Println("set alarm:", al.Name)
}

func (svc *AlarmSvc) Put(msg *cmd.Msg) {
	svc.in <- msg
}

func (svc *AlarmSvc) matchMsgs() {
	fmt.Println("msg-matching gopher started")
	for msg := range svc.in {
		t := msg.T.AsTime().UnixMicro()
		svc.Alarms.Range(func(key, value interface{}) bool {
			// Type assertion
			al, ok := value.(*Alarm)
			if !ok {
				return true // continue iteration
			}
			al.Events.Store(t, &Event{
				Msg:      msg,
				Occurred: time.Now(),
			})
			al.EventCount.Add(1)
			if al.EventCount.Load() >= al.Threshold {
				// fire at most once per period
				if al.LastTriggered.After(time.Now().Add(-al.Period)) {
					return true
				}
				svc.triggered <- al
			}

			return true // continue iteration
		})
	}
}

func (svc *AlarmSvc) kickOldMatchedEvents() {
	for {
		svc.Alarms.Range(func(key, value interface{}) bool {
			al, ok := value.(*Alarm) // Type assertion
			if !ok {
				return true // continue iteration
			}
			al.Events.Range(func(i, ev interface{}) bool {
				event, ok := ev.(*Event) // Type assertion
				if !ok {
					return true // continue iteration
				}
				if event.Occurred.Before(time.Now().Add(-al.Period)) {
					al.Events.Delete(i)
					al.EventCount.Add(-1)
				}
				return true // continue iteration
			})
			return true // continue iteration
		})
		time.Sleep(time.Second * 10)
	}
}

// callActions of triggered alarms
func (svc *AlarmSvc) callActions() {
	for a := range svc.triggered {
		fmt.Println("alarm triggered:", a.Name)
		a.LastTriggered = time.Now()
		// prevent immediate firing
		if time.Now().Before(svc.started.Add(a.Period)) {
			fmt.Println("skipped action, app just started")
			continue
		}
		err := a.Action()
		if err != nil {
			fmt.Println("alarm action err:", err)
		}
		// reset events & event count
		a.Events = sync.Map{}
		a.EventCount.Store(0)
	}
}
