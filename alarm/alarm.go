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
	In        chan *cmd.Msg // buffer doesn't help
	triggered chan *Alarm   // buffer doesn't help
	Alarms    sync.Map
}

type Alarm struct {
	Name          string
	Match         func(*cmd.Msg) bool
	Recent        *ring.RingBuffer
	Period        time.Duration // period of analysis
	Threshold     int32
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
		In:        make(chan *cmd.Msg, 100),
		triggered: make(chan *Alarm),
	}
	// we need some gophers
	// adding more gophers matching messages doesn't help
	for i := 0; i < 4; i++ {
		go s.matchMsgs()
	}
	go s.kickOldEvents()
	go s.callActions()
	return s
}

func (svc *AlarmSvc) Set(al *Alarm) {
	svc.Alarms.Store(al.Name, al)
	fmt.Println("set alarm:", al.Name)
}

func (svc *AlarmSvc) matchMsgs() {
	fmt.Println("alarm-matching gopher started")
	for msg := range svc.In {
		t := msg.T.AsTime().UnixMicro()
		svc.Alarms.Range(func(key, value interface{}) bool {
			al, ok := value.(*Alarm) // Type assertion
			if !ok {
				return true // continue iteration
			}
			// Use al
			if !al.Match(msg) {
				return true // continue iteration
			}
			al.Events.Store(t, &Event{
				Msg:      msg,
				Occurred: time.Now(),
			})
			al.EventCount.Add(1)
			if al.EventCount.Load() >= al.Threshold {
				if al.LastTriggered.After(time.Now().Add(-al.Period)) {
					return true
				}
				svc.triggered <- al
			}

			return true // continue iteration
		})
	}
}

func (svc *AlarmSvc) kickOldEvents() {
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

// callActions is a gopher that listens for triggered alarms
func (svc *AlarmSvc) callActions() {
	for a := range svc.triggered {
		fmt.Println("alarm triggered:", a.Name)
		a.LastTriggered = time.Now()
		err := a.Action()
		if err != nil {
			fmt.Println("alarm action err:", err)
		}
		// reset events & event count
		a.Events = sync.Map{}
		a.EventCount.Store(0)
	}
}
