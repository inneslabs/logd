package ring

import "sync/atomic"

type RingBuffer struct {
	head   int
	size   int
	values []*[]byte
	Writes atomic.Uint64
}

// NewRingBuffer returns a pointer to a new RingBuffer of given size
func NewRingBuffer(size int) *RingBuffer {
	r := &RingBuffer{
		size:   size,
		values: make([]*[]byte, size),
	}
	return r
}

func (r *RingBuffer) Size() int {
	return r.size
}

func (r *RingBuffer) Write(data *[]byte) {
	r.Writes.Add(uint64(1))
	r.values[r.head] = data
	r.head = (r.head + 1) % r.size
}

// Read returns limit of data
func (r *RingBuffer) Read(offset, limit int) []*[]byte {
	if limit > r.size || limit < 0 {
		limit = r.size
	}
	output := make([]*[]byte, 0)
	reads := 0
	index := (r.head + (r.size - 1) + offset) % r.size
	for reads < limit {
		if r.values[index] != nil {
			output = append(output, r.values[index])
		}
		index--
		if index < 0 {
			index = r.size - 1
		}
		reads++
	}
	return output
}
