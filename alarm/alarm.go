/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package alarm

import (
	"fmt"
	"sync"
	"time"

	"github.com/swissinfo-ch/logd/cmd"
	"github.com/swissinfo-ch/logd/ring"
)

type Svc struct {
	In        chan *cmd.Msg // buffer doesn't help
	triggered chan *Alarm   // buffer doesn't help
	alarms    map[string]*Alarm
	mu        sync.Mutex
}

type Alarm struct {
	Name          string
	Match         func(*cmd.Msg) bool
	Recent        *ring.RingBuffer
	Period        time.Duration // period of analysis
	Threshold     int
	Events        map[int64]*Event // key is unix milli
	Action        func() error
	LastTriggered time.Time
	mu            sync.Mutex
}

type Event struct {
	Msg      *cmd.Msg
	Occurred time.Time
}

func NewSvc() *Svc {
	s := &Svc{
		In:        make(chan *cmd.Msg),
		triggered: make(chan *Alarm),
		alarms:    make(map[string]*Alarm),
	}
	// we need some gophers
	// adding more gophers matching messages doesn't help
	go s.matchMsgs()
	go s.kickOldEvents()
	go s.callActions()
	return s
}

func (s *Svc) Set(al *Alarm) {
	s.mu.Lock()
	s.alarms[al.Name] = al
	s.alarms[al.Name].Events = make(map[int64]*Event)
	s.mu.Unlock()
	fmt.Println("set alarm:", al.Name)
}

func (s *Svc) GetAll() map[string]*Alarm {
	return s.alarms
}

func (s *Svc) matchMsgs() {
	fmt.Println("alarm-matching gopher started")
	for msg := range s.In {
		for _, al := range s.alarms {
			if !al.Match(msg) {
				continue
			}
			al.Events[msg.T.AsTime().UnixMicro()] = &Event{
				Msg:      msg,
				Occurred: time.Now(),
			}
			if len(al.Events) >= al.Threshold {
				if al.LastTriggered.After(time.Now().Add(-al.Period)) {
					continue
				}
				s.triggered <- al
				al.mu.Lock()
				al.Events = make(map[int64]*Event)
				al.LastTriggered = time.Now()
				al.mu.Unlock()
			}
		}
	}
}

func (s *Svc) kickOldEvents() {
	for {
		for _, al := range s.alarms {
			for i, ev := range al.Events {
				if ev.Occurred.Before(time.Now().Add(-al.Period)) {
					al.mu.Lock()
					delete(al.Events, i)
					al.mu.Unlock()
				}
			}
		}
		time.Sleep(time.Second)
	}
}

func (s *Svc) callActions() {
	for a := range s.triggered {
		fmt.Println("alarm triggered:", a.Name)
		err := a.Action()
		if err != nil {
			fmt.Println("alarm action err:", err)
		}
	}
}
