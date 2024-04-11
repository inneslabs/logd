package pkg

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

type Pkg struct {
	Sum, TimeBytes, Payload []byte
}

// Unpack unpacks data into pkg.
// This approach allows caller to allocate pkg efficiently.
func Unpack(data []byte, pkg *Pkg) error {
	if len(data) < 32+15 {
		return errors.New("data too short")
	}
	pkg.Sum = data[:32]              /*32B sha256*/
	pkg.TimeBytes = data[32 : 32+15] /*15B time*/
	pkg.Payload = data[32+15:]
	return nil
}

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

func Verify(secret []byte, ttl time.Duration, p *Pkg) (bool, error) {
	var t time.Time
	err := t.UnmarshalBinary(p.TimeBytes)
	if err != nil {
		return false, fmt.Errorf("err unmarshaling time: %w", err)
	}
	if t.After(time.Now()) || t.Before(time.Now().Add(-ttl)) {
		return false, errors.New("time is outside of threshold")
	}
	totalLen := len(secret) + len(p.TimeBytes) + len(p.Payload)
	data := make([]byte, 0, totalLen)
	data = append(data, secret...)
	data = append(data, p.TimeBytes...)
	data = append(data, p.Payload...)
	h := sha256.Sum256(data)
	return bytes.Equal(p.Sum, h[:32]), nil
}
