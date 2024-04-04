package guard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/inneslabs/logd/pkg"
	cuckoo "github.com/seiflotfy/cuckoofilter"
)

type Guard struct {
	filter    *cuckoo.Filter
	packetTtl time.Duration
	quit      chan struct{}
}

type Cfg struct {
	FilterCap uint          `yaml:"filter_cap"` // Filter capacity
	FilterTtl time.Duration `yaml:"filter_ttl"` // Reset filter after
	PacketTtl time.Duration `yaml:"packet_ttl"` // Packet validity
}

func NewGuard(ctx context.Context, cfg *Cfg) *Guard {
	g := &Guard{
		filter:    cuckoo.NewFilter(cfg.FilterCap),
		packetTtl: cfg.PacketTtl,
		quit:      make(chan struct{}),
	}
	go func() {
		done := ctx.Done()
		for {
			select {
			case <-done:
				g.quit <- struct{}{}
				return
			case <-time.After(cfg.FilterTtl):
				g.filter.Reset()
			}
		}
	}()
	return g
}

func (g *Guard) Good(secret []byte, p *pkg.Pkg) bool {
	authed, err := g.verify(secret, p)
	if err != nil || !authed {
		fmt.Printf("unauthorised: err: %v\n", err)
		return false
	}
	if g.replay(p.Sum) {
		if rand.Intn(1000) < 1 {
			fmt.Println("~1000 replays detected")
		}
		return false
	}
	return true
}

func (g *Guard) Quit() <-chan struct{} {
	return g.quit
}

func (g *Guard) replay(sum []byte) bool {
	return !g.filter.InsertUnique(sum)
}

func (g *Guard) verify(secret []byte, p *pkg.Pkg) (bool, error) {
	var t time.Time
	err := t.UnmarshalBinary(p.TimeBytes)
	if err != nil {
		return false, fmt.Errorf("convert bytes to time err: %w", err)
	}
	if t.After(time.Now()) || t.Before(time.Now().Add(-g.packetTtl)) {
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
