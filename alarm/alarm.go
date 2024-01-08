package alarm

import (
	"fmt"
	"sync"
	"time"

	"github.com/swissinfo-ch/logd/msg"
	"github.com/swissinfo-ch/logd/pack"
	"github.com/swissinfo-ch/logd/transport"
)

type Svc struct {
	In        chan *[]byte
	Alarms    map[string]*Alarm
	triggered chan *Alarm
	mu        sync.Mutex
}

type Alarm struct {
	Name          string
	Match         func(*msg.Msg) bool
	Period        time.Duration // period of analysis
	Threshold     int
	Events        map[int64]*Event // key is unix milli
	Action        func() error
	LastTriggered time.Time
	mu            sync.Mutex
}

type Event struct {
	Msg      *msg.Msg
	Occurred time.Time
}

func NewSvc() *Svc {
	s := &Svc{
		In:        make(chan *[]byte, transport.ChanBufferSize),
		Alarms:    make(map[string]*Alarm),
		triggered: make(chan *Alarm),
	}
	go s.matchMsgs()
	go s.kickOldEvents()
	go s.callActions()
	return s
}

func (s *Svc) Set(al *Alarm) {
	s.mu.Lock()
	s.Alarms[al.Name] = al
	s.Alarms[al.Name].Events = make(map[int64]*Event)
	s.mu.Unlock()
	fmt.Println("set alarm:", al.Name)
}

func (s *Svc) matchMsgs() {
	for data := range s.In {
		m, err := pack.UnpackMsg(*data)
		if err != nil {
			fmt.Println("alarm unpack msg err:", err)
		}
		for _, al := range s.Alarms {
			if !al.Match(m) {
				continue
			}
			fmt.Println("event matched alarm", al.Name)
			al.Events[m.Timestamp] = &Event{
				Msg:      m,
				Occurred: time.Now(),
			}
			if len(al.Events) >= al.Threshold {
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
		for _, al := range s.Alarms {
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
		if a.LastTriggered.After(time.Now().Add(-a.Period)) {
			return
		}
		fmt.Println("alarm triggered:", a.Name)
		err := a.Action()
		if err != nil {
			fmt.Println("alarm action err:", err)
		}
	}
}
