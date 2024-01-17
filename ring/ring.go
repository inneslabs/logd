/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package ring

import (
	"bufio"
	"io"
	"sync/atomic"
)

type RingBuffer struct {
	head   atomic.Uint32
	size   uint32
	values [][]byte
	Writes atomic.Uint64
}

const (
	zero = uint32(0)
	one  = uint64(1)
)

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

func (b *RingBuffer) Write(data []byte) {
	b.Writes.Add(one)
	head := b.head.Load()
	b.values[head] = data
	b.head.Store((head + 1) % b.size)
}

// Read returns limit of data
func (b *RingBuffer) Read(offset, limit uint32) [][]byte {
	if limit > b.size || limit < zero {
		limit = b.size
	}
	output := make([][]byte, 0)
	reads := zero
	head := b.head.Load()
	index := (head + (b.size - 1) + offset) % b.size
	for reads < limit {
		if b.values[index] != nil {
			output = append(output, b.values[index])
		}
		if index == zero {
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

func (b *RingBuffer) ScanFrom(r io.Reader) error {
	s := bufio.NewScanner(r)
	for s.Scan() {
		b.Write(s.Bytes())
	}
	return s.Err()
}
