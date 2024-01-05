package logdutil

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

func AuthMsg(secret, payload []byte) ([]byte, error) {
	h := sha256.New()
	_, err := h.Write(payload)
	if err != nil {
		return nil, fmt.Errorf("err writing payload to hash: %w", err)
	}
	_, err = h.Write(secret)
	if err != nil {
		return nil, fmt.Errorf("err writing secret to hash: %w", err)
	}
	buf := &bytes.Buffer{}
	_, err = buf.Write(h.Sum(nil))
	if err != nil {
		return nil, fmt.Errorf("err writing sum to buffer: %w", err)
	}
	_, err = buf.Write(payload)
	if err != nil {
		return nil, fmt.Errorf("err writing payload to buffer: %w", err)
	}
	return buf.Bytes(), nil
}
