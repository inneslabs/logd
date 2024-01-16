/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package ring

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestReadWhenEmpty(t *testing.T) {
	r := NewRingBuffer(5)
	n := len(r.Read(0, 6))
	if n > 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestWriteAndReadFull(t *testing.T) {
	if !testWriteAndRead(10, 10) {
		t.FailNow()
	}
}

func TestWriteAndReadOver(t *testing.T) {
	if !testWriteAndRead(10, 15) {
		t.FailNow()
	}
}

func TestOffset(t *testing.T) {
	testWriteAndReadWithOffset(10, 2, 1)
	testWriteAndReadWithOffset(10, 5, 3)
	testWriteAndReadWithOffset(10, 10, 5)
	testWriteAndReadWithOffset(10, 20, 10)
}

func testWriteAndReadWithOffset(size, nWrites, offset int) bool {
	r := NewRingBuffer(uint32(size))
	writes := make([][]byte, 0)
	for i := 0; i < nWrites; i++ {
		buf := make([]byte, 32)
		rand.Read(buf)
		writes = append(writes, buf)
		r.Write(buf)
	}
	items := r.Read(uint32(offset), uint32(nWrites))
	equal := true
	for i := 0; i < size; i++ {
		a := items[i]
		b := writes[(((len(writes)-1)-i)+offset)%len(writes)]
		if !bytes.Equal(a, b) {
			equal = false
			break
		}
	}
	return equal
}

func testWriteAndRead(size, nWrites int) bool {
	r := NewRingBuffer(uint32(size))
	writes := make([][]byte, 0)
	for i := 0; i < nWrites; i++ {
		buf := make([]byte, 32)
		rand.Read(buf)
		writes = append(writes, buf)
		r.Write(buf)
	}
	items := r.Read(0, uint32(nWrites))
	equal := true
	for i := 0; i < size; i++ {
		a := items[i]
		b := writes[(len(writes)-1)-i]
		if !bytes.Equal(a, b) {
			equal = false
			break
		}
	}
	return equal
}
