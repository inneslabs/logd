package auth

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

const (
	hashLen = 32
	sigTtl  = time.Millisecond * 500
	sumLen  = 32
	timeLen = 8
)

func Sign(secret, payload []byte, t time.Time) ([]byte, error) {
	timeBytes, err := convertTimeToBytes(t)
	if err != nil {
		return nil, fmt.Errorf("convert time to bytes err: %w", err)
	}
	// concat secret, timeBytes & payload
	data := append(secret, timeBytes...)
	data = append(data, payload...)
	h := sha256.Sum256(data)
	sum := h[:hashLen]
	// return sum + time + payload
	signed := make([]byte, 0, hashLen+timeLen+len(payload))
	signed = append(signed, sum...)
	signed = append(signed, timeBytes...)
	return append(signed, payload...), nil
}

func Verify(secret, sum, timeBytes, payload []byte) (bool, error) {
	t, err := convertBytesToTime(timeBytes)
	if err != nil {
		return false, fmt.Errorf("convert bytes to time err: %w", err)
	}
	if t.After(time.Now().Add(sigTtl)) ||
		t.Before(time.Now().Add(-sigTtl)) {
		return false, errors.New("time is outside of threshold")
	}
	data := append(secret, timeBytes...)
	data = append(data, payload...)
	h := sha256.Sum256(data)
	mySum := h[:hashLen]
	return bytes.Equal(sum, mySum), nil
}
