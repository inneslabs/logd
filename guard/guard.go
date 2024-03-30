package guard

import (
	"bytes"
	"container/ring"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/inneslabs/logd/pkg"
)

type Guard struct {
	history *ring.Ring
	mutex   sync.Mutex
	sumTtl  time.Duration
}

type Cfg struct {
	HistorySize int           `yaml:"history_size"`
	SumTtl      time.Duration `yaml:"sum_ttl"`
}

func NewGuard(cfg *Cfg) *Guard {
	return &Guard{
		history: ring.New(cfg.HistorySize),
		sumTtl:  cfg.SumTtl,
	}
}

func (g *Guard) Good(secret []byte, p *pkg.Pkg) bool {
	authed, err := g.verify(secret, p)
	if err != nil || !authed {
		return false
	}
	return !g.replay(p.Sum)
}

func (g *Guard) replay(sum []byte) bool {
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
	return found
}

func (g *Guard) verify(secret []byte, p *pkg.Pkg) (bool, error) {
	var t time.Time
	err := t.UnmarshalBinary(p.TimeBytes)
	if err != nil {
		return false, fmt.Errorf("convert bytes to time err: %w", err)
	}
	if t.After(time.Now()) || t.Before(time.Now().Add(-g.sumTtl)) {
		return false, errors.New("time is outside of threshold")
	}
	totalLen := len(secret) + len(p.TimeBytes) + len(p.Payload)
	data := make([]byte, 0, totalLen)
	data = append(data, secret...)
	data = append(data, p.TimeBytes...)
	data = append(data, p.Payload...)
	h := sha256.Sum256(data)
	return bytes.Equal(p.Sum, h[:32]), nil
}
