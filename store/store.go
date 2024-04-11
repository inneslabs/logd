package store

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/intob/logd/ring"
)

type Store struct {
	rings    map[string]*ring.Ring
	fallback *ring.Ring
	nWrites  atomic.Uint64
}

type Cfg struct {
	RingSizes    map[string]uint32 `yaml:"ring_sizes"`
	FallbackSize uint32            `yaml:"fallback_size"`
}

func NewStore(cfg *Cfg) *Store {
	s := &Store{
		rings:    make(map[string]*ring.Ring, len(cfg.RingSizes)),
		fallback: ring.NewRing(cfg.FallbackSize),
	}
	for key, size := range cfg.RingSizes {
		s.rings[key] = ring.NewRing(size)
	}
	return s
}

// Write writes to the ring of key, or fallback ring
func (s *Store) Write(key string, data []byte) {
	s.nWrites.Add(uint64(1))
	part, ok := s.rings[key]
	if !ok {
		s.fallback.Write(data)
		return
	}
	part.Write(data)
}

func (s *Store) HeadsAndSizes() map[string][2]uint32 {
	info := make(map[string][2]uint32, len(s.rings)+1)
	for key, ring := range s.rings {
		info[key] = [2]uint32{ring.Head(), ring.Size()}
	}
	info["_fallback"] = [2]uint32{s.fallback.Head(), s.fallback.Size()}
	return info
}

// Read reads up to limit items, from offset,
// all rings with the given key prefix
func (s *Store) Read(keyPrefix string, offset, limit uint32) <-chan []byte {
	out := make(chan []byte, 1)
	go func() {
		defer close(out)
		exactRing := s.rings[keyPrefix]
		if exactRing != nil {
			for d := range exactRing.Read(offset, limit) {
				out <- d
			}
			return
		}
		var count uint32
		var matchedPrefix bool
		for key, r := range s.rings {
			if strings.HasPrefix(key, keyPrefix) {
				matchedPrefix = true
				fmt.Println("reading from", key)
				for d := range r.Read(offset, limit-count) {
					out <- d
					count++
					if count >= limit {
						return
					}
				}
			}
		}
		if !matchedPrefix {
			fmt.Println("reading from fallback")
			for d := range s.fallback.Read(offset, limit) {
				out <- d
			}
		}
	}()
	return out
}

func (s *Store) NWrites() uint64 {
	return s.nWrites.Load()
}
