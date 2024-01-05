package transport

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
)

const SUM_LEN = 32

func unpackMsg(msg []byte) (sum, payload []byte, err error) {
	if len(msg) < SUM_LEN {
		return nil, nil, errors.New("msg too short")
	}
	return msg[:SUM_LEN], msg[SUM_LEN:], err
}

func validateSum(secret, sum, payload []byte) (bool, error) {
	h := sha256.New()
	_, err := h.Write(payload)
	if err != nil {
		return false, fmt.Errorf("write payload: %w", err)
	}
	h.Write(secret)
	if err != nil {
		return false, fmt.Errorf("write token: %w", err)
	}
	mySum := h.Sum(nil)
	return bytes.Equal(sum, mySum), nil
}
