package guard

import (
	"bytes"
	"container/ring"
	"sync"
)

type Guard struct {
	history *ring.Ring
	mutex   sync.Mutex
}

type Cfg struct {
	HistorySize int `yaml:"history_size"`
}

func NewGuard(cfg *Cfg) *Guard {
	return &Guard{
		history: ring.New(cfg.HistorySize),
	}
}

func (g *Guard) Good(sum []byte) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	found := false
	g.history.Do(func(v interface{}) {
		b, ok := v.([]byte)
		if !ok {
			return
		}
		if bytes.Equal(b, sum) {
			found = true
		}
	})
	if !found {
		g.history.Value = sum
		g.history = g.history.Next()
	}
	return !found
}
