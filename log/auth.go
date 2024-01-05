package log

import (
	"crypto/sha256"
	"encoding/binary"
	"time"
)

func Sign(secret, payload []byte, t time.Time) []byte {
	// Convert the time to a byte slice.
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(t.Unix()))

	// Concat secret, time, and payload
	data := append(secret, timeBytes...)
	data = append(data, payload...)

	// Compute SHA256
	h := sha256.Sum256(data)
	return h[:32]
}
