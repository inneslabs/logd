package alarm

import (
	"time"

	"github.com/swissinfo-ch/logd/msg"
)

type Alarm struct {
	Match     func(*msg.Msg) bool
	Period    time.Duration // period of analysis
	Threshold int
	Events    map[int64]*Event // key is unix milli
}

type Event struct {
	Msg     *msg.Msg
	Occured time.Time
}

type AlarmService struct {
	Alarms map[string]*Alarm
	In     chan *[]byte
}
