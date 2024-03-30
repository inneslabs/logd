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
		reads := uint32(0)
		head := b.head.Load()

		var index uint32
		// Calculate start index based on offset, ensuring it wraps correctly
		if head < offset {
			index = b.size - (offset - head)
		} else {
			index = head - offset - 1
		}

		// Adjust if we're starting beyond the most recent write
		if index == b.size {
			index = 0
		}

		// Loop to send data through the channel
		for reads < limit {
			// Calculate the correct index to read from, wrapping around if necessary
			readIndex := (index - reads) % b.size
			if b.values[readIndex] == nil {
				break
			}
			ch <- b.values[readIndex]
			reads++
			// Stop if we've looped back to the head
			if reads >= limit || readIndex == head {
				break
			}
		}
	}()

	return ch
}

func (b *Ring) Head() uint32 {
	return b.head.Load()
}
