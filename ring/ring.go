/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package ring

import (
	"sync/atomic"
)

type RingBuffer struct {
	head      atomic.Uint32
	size      uint32
	values    [][]byte
	numWrites atomic.Uint64
}

// NewRingBuffer returns a pointer to a new RingBuffer of given size
func NewRingBuffer(size uint32) *RingBuffer {
	r := &RingBuffer{
		size:   size,
		values: make([][]byte, size),
	}
	return r
}

func (b *RingBuffer) Size() uint32 {
	return b.size
}

func (b *RingBuffer) NumWrites() uint64 {
	return b.numWrites.Load()
}

// Write writes the data to buffer at position of head,
// head is then atomically incremented
func (b *RingBuffer) Write(data []byte) {
	b.numWrites.Add(uint64(1))
	head := b.head.Load()
	b.values[head] = data
	b.head.Store((head + 1) % b.size)
}

// Read returns limit of data
func (b *RingBuffer) Read(offset, limit uint32) [][]byte {
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

func (b *RingBuffer) Head() uint32 {
	return b.head.Load()
}

// Returns the record {index} slots ahead of head (oldest first)
func (b *RingBuffer) ReadOne(index uint32) []byte {
	return b.values[index%b.size]
}
