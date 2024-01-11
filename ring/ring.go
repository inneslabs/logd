/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package ring

import (
	"sync/atomic"
)

type RingBuffer struct {
	head   atomic.Uint32
	size   uint32
	values []*[]byte
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
		values: make([]*[]byte, size),
	}
	return r
}

func (r *RingBuffer) Size() uint32 {
	return r.size
}

func (r *RingBuffer) Write(data *[]byte) {
	r.Writes.Add(one)
	head := r.head.Load()
	r.values[head] = data
	r.head.Store((head + 1) % r.size)
}

// Read returns limit of data
func (r *RingBuffer) Read(offset, limit uint32) []*[]byte {
	if limit > r.size || limit < zero {
		limit = r.size
	}
	output := make([]*[]byte, 0)
	reads := zero
	head := r.head.Load()
	index := (head + (r.size - 1) + offset) % r.size
	for reads < limit {
		if r.values[index] != nil {
			output = append(output, r.values[index])
		}
		if index == zero {
			index = r.size
		}
		index--
		reads++
	}
	return output
}

func (r *RingBuffer) Head() uint32 {
	return r.head.Load()
}

// Returns the record {offset} slots ahead of head (oldest first)
func (r *RingBuffer) ReadOne(index uint32) *[]byte {
	return r.values[index%r.size]
}
