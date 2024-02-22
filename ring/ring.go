/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package ring

import (
	"sync/atomic"
)

type Ring struct {
	head   atomic.Uint32
	size   uint32
	values [][]byte
}

// NewRingBuffer returns a pointer to a new RingBuffer of given size
func NewRing(size uint32) *Ring {
	r := &Ring{
		size:   size,
		values: make([][]byte, size),
	}
	return r
}

func (b *Ring) Size() uint32 {
	return b.size
}

// Write writes the data to buffer at position of head,
// head is then atomically incremented
func (b *Ring) Write(data []byte) {
	head := b.head.Load()
	b.values[head] = data
	b.head.Store((head + 1) % b.size)
}

// Read returns limit of data
func (b *Ring) Read(offset, limit uint32) [][]byte {
	if limit > b.size {
		limit = b.size
	}
	// pre-allocating the output slice is 2x faster
	output := make([][]byte, 0, limit)
	reads := uint32(0)
	head := b.head.Load()
	index := (head + (b.size - 1) + offset) % b.size
	for reads < limit {
		if b.values[index] != nil {
			output = append(output, b.values[index])
		}
		if index == 0 {
			index = b.size
		}
		index--
		reads++
	}
	return output
}

func (b *Ring) Head() uint32 {
	return b.head.Load()
}

// Returns the record {index} slots ahead of head (oldest first)
func (b *Ring) ReadOne(index uint32) []byte {
	return b.values[index%b.size]
}
