package sign

import (
	"crypto/sha256"
	"time"
)

func Sign(secret, payload []byte) []byte {
	timeBytes, _ := time.Now().MarshalBinary()
	// pre-allocate slice
	totalLen := 32 + 8 + len(payload)
	data := make([]byte, 0, totalLen)
	// copy data
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	// compute checksum
	h := sha256.Sum256(data)
	// append sum and timeBytes to data slice
	data = append(data[:0], h[:32]...)
	data = append(data, timeBytes...)
	return append(data, payload...)
}
