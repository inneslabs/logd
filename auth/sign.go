package auth

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

const (
	SigTtl  = time.Millisecond * 100
	SumLen  = 32
	TimeLen = 8
)

// Sign payload using current time + sig ttl
func Sign(secret, payload []byte) ([]byte, error) {
	t := time.Now().Add(SigTtl)
	return SignWithTime(secret, payload, t)
}

// Sign payload with given time
func SignWithTime(secret, payload []byte, t time.Time) ([]byte, error) {
	timeBytes, err := convertTimeToBytes(t)
	if err != nil {
		return nil, fmt.Errorf("convert time to bytes err: %w", err)
	}
	// pre-allocate slice
	totalLen := SumLen + len(timeBytes) + len(payload)
	data := make([]byte, 0, totalLen)
	// copy data
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	// compute checksum
	h := sha256.Sum256(data)
	// append sum and timeBytes to data slice
	data = append(data[:0], h[:SumLen]...)
	data = append(data, timeBytes...)
	return append(data, payload...), nil
}

// Verify signed payload
func Verify(secret []byte, unpk *Unpacked) (bool, error) {
	// if secret is unset, return true immediately
	if len(secret) == 0 {
		return true, nil
	}
	// convert time
	t, err := convertBytesToTime(unpk.TimeBytes)
	if err != nil {
		return false, fmt.Errorf("convert bytes to time err: %w", err)
	}
	// verify timestamp is within threshold
	if t.After(time.Now().Add(SigTtl)) ||
		t.Before(time.Now().Add(-SigTtl)) {
		return false, errors.New("time is outside of threshold")
	}
	// pre-allocate slice
	totalLen := len(secret) + len(unpk.TimeBytes) + len(unpk.Payload)
	data := make([]byte, 0, totalLen)
	// copy data
	data = append(data, secret...)
	data = append(data, unpk.TimeBytes...)
	data = append(data, unpk.Payload...)
	// compute checksum
	h := sha256.Sum256(data)
	// verify equality
	return bytes.Equal(unpk.Sum, h[:SumLen]), nil
}
