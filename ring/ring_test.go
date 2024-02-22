/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package ring

import (
	"fmt"
	"testing"
)

func TestReadWhenEmpty(t *testing.T) {
	r := NewRing(5)
	n := len(r.Read(0, 6))
	if n > 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

// BenchmarkWriteRingBuffer tests the performance of writing to the RingBuffer
func BenchmarkWriteRingBuffer(b *testing.B) {
	buffer := NewRing(1024)       // Adjust size as needed
	data := []byte("sample data") // Sample data to write

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Write(data)
	}
}

// BenchmarkReadRingBuffer tests the performance of reading from the RingBuffer
func BenchmarkReadRingBuffer(b *testing.B) {
	size := uint32(4096)
	buf := NewRing(size)
	for i := uint32(0); i < size; i++ {
		data := []byte(fmt.Sprintf("sample data %d", i))
		buf.Write(data)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Read(uint32(i)%size, 512)
	}
}
