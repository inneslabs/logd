package store

import (
	"strings"
	"sync/atomic"

	"github.com/swissinfo-ch/logd/ring"
)

type Store struct {
	rings     map[string]*ring.Ring
	fallback  *ring.Ring
	numWrites atomic.Uint64
}

type Cfg struct {
	RingSizes    map[string]uint32
	FallbackSize uint32
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
	s.numWrites.Add(uint64(1))
	part := s.rings[key]
	if part == nil {
		s.fallback.Write(data)
		return
	}
	part.Write(data)
}

// Read reads up to limit items, from offset,
// all rings with the given key prefix
func (s *Store) Read(keyPrefix string, offset, limit uint32) <-chan []byte {
	out := make(chan []byte)
	go func() {
		// try to read from exact ring
		exactRing := s.rings[keyPrefix]
		if exactRing != nil {
			readRing(exactRing, offset, limit, out)
			close(out)
			return
		}
		// fallback to prefix, ranging through rings
		var count uint32
		for key, r := range s.rings {
			if strings.HasPrefix(key, keyPrefix) {
				count += readRing(r, offset, limit-count, out)
				if count >= limit {
					close(out)
					return
				}
			}
		}
		readRing(s.fallback, offset, limit-count, out)
		close(out)
	}()
	return out
}

func (s *Store) NumWrites() uint64 {
	return s.numWrites.Load()
}

// readPart reads up to limit items from offset from given part,
// sending on given channel, returning count sent
func readRing(r *ring.Ring, offset, limit uint32, out chan []byte) uint32 {
	var count uint32
	data := r.Read(offset, limit)
	for _, d := range data {
		out <- d
		count++
	}
	return count
}
