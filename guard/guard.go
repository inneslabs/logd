package guard

import (
	"bytes"
	"container/ring"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/inneslabs/logd/sign"
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

func (g *Guard) Good(secret []byte, pkg *sign.Pkg) bool {
	authed, err := g.verify(secret, pkg)
	if err != nil || !authed {
		return false
	}
	return !g.replay(pkg.Sum)
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

// Verify signed payload
func (g *Guard) verify(secret []byte, pkg *sign.Pkg) (bool, error) {
	// convert time
	t, err := convertBytesToTime(pkg.TimeBytes)
	if err != nil {
		return false, fmt.Errorf("convert bytes to time err: %w", err)
	}
	// verify timestamp is within threshold
	if t.After(time.Now()) || t.Before(time.Now().Add(-g.sumTtl)) {
		return false, errors.New("time is outside of threshold")
	}
	// pre-allocate slice
	totalLen := len(secret) + len(pkg.TimeBytes) + len(pkg.Payload)
	data := make([]byte, 0, totalLen)
	// copy data
	data = append(data, secret...)
	data = append(data, pkg.TimeBytes...)
	data = append(data, pkg.Payload...)
	// compute checksum
	h := sha256.Sum256(data)
	// verify equality
	return bytes.Equal(pkg.Sum, h[:32]), nil
}

func convertBytesToTime(b []byte) (time.Time, error) {
	if len(b) != 8 {
		return time.Time{}, fmt.Errorf("byte slice must be exactly 8 bytes long")
	}
	nano, err := bytesToInt64(b)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to convert bytes to int64: %w", err)
	}
	return time.Unix(0, nano), nil
}

func bytesToInt64(b []byte) (int64, error) {
	if len(b) != 8 {
		return 0, fmt.Errorf("byte slice must be exactly 8 bytes long")
	}

	var num int64
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.BigEndian, &num)
	if err != nil {
		return 0, err
	}
	return num, nil
}
