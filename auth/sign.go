package auth

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/swissinfo-ch/logd/msg"
)

const (
	timeLen       = 8
	hashLen       = 32
	timeThreshold = time.Millisecond * 50
)

func Sign(secret []byte, msg *msg.Msg, t time.Time) ([]byte, error) {
	timeBytes, err := convertTimeToBytes(t)
	if err != nil {
		return nil, fmt.Errorf("convert time to bytes err: %w", err)
	}
	payload, err := cbor.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("cbor marshal err: %w", err)
	}
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
	if t.After(time.Now().Add(timeThreshold)) || t.Before(time.Now().Add(-timeThreshold)) {
		return false, errors.New("time is outside of threshold")
	}
	data := append(secret, timeBytes...)
	data = append(data, payload...)
	h := sha256.Sum256(data)
	mySum := h[:hashLen]
	return bytes.Equal(sum, mySum), nil
}
