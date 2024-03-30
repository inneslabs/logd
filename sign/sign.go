package sign

import (
	"crypto/sha256"
	"time"
)

func Sign(secret, payload []byte) []byte {
	timeBytes, _ := time.Now().MarshalBinary() // 15B
	data := make([]byte, 0, 32 /* sha256 */ +15 /* time */ +len(payload))
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	h := sha256.Sum256(data)
	data = append(data[:0], h[:32]...)
	data = append(data, timeBytes...)
	return append(data, payload...)
}
