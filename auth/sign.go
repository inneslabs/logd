/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package auth

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

const (
	sigTtl  = time.Millisecond * 500
	SumLen  = 32
	TimeLen = 8
)

func Sign(secret, payload []byte, t time.Time) ([]byte, error) {
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
	sum := h[:SumLen]
	// return sum + time + payload
	signed := make([]byte, 0, SumLen+TimeLen+len(payload))
	signed = append(signed, sum...)
	signed = append(signed, timeBytes...)
	return append(signed, payload...), nil
}

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
	if t.After(time.Now().Add(sigTtl)) ||
		t.Before(time.Now().Add(-sigTtl)) {
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
