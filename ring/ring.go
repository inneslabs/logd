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

func (b *Ring) Read(offset, limit uint32) <-chan []byte {
	ch := make(chan []byte, limit) // Buffered channel with size 'limit'

	go func() {
		defer close(ch) // Ensure channel is closed when operation completes

		if limit > b.size {
			limit = b.size
		}

		head := b.head.Load()

		// Calculate the actual number of written entries to avoid reading uninitialized data
		written := head
		if written > b.size {
			written = b.size
		}

		// Calculate start index based on offset, ensuring it wraps correctly
		var index uint32
		if written <= offset {
			// If offset is beyond what's written, don't read anything
			return
		} else {
			// Adjust index to start reading from the correct position
			index = (head + b.size - offset) % b.size
			if index == 0 && offset > 0 {
				// Special case when offset equals the size
				index = b.size - 1
			} else if index > 0 {
				// Normally, decrement to get the correct starting position
				index -= 1
			}
		}

		reads := uint32(0)
		// Loop to send data through the channel, reading forwards
		for reads < limit && reads < written {
			readIndex := (index + reads) % b.size
			ch <- b.values[readIndex]
			reads++
		}
	}()

	return ch
}

func (b *Ring) Head() uint32 {
	return b.head.Load()
}

// Returns the record {index} slots ahead of head (oldest first)
func (b *Ring) ReadOne(index uint32) []byte {
	return b.values[index%b.size]
}
