package guard

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/intob/logd/pkg"
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
	authed, err := pkg.Verify(secret, g.packetTtl, p)
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
