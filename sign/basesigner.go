package sign

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

type BaseSigner struct {
	sumTtl time.Duration
}

type BaseSignerCfg struct {
	SumTtl time.Duration `yaml:"sum_ttl"`
}

func NewBaseSigner(cfg *BaseSignerCfg) *BaseSigner {
	return &BaseSigner{
		sumTtl: cfg.SumTtl,
	}
}

func (s *BaseSigner) Sign(secret, payload []byte) ([]byte, error) {
	return s.signWithTime(secret, payload, time.Now())
}

func (s *BaseSigner) signWithTime(secret, payload []byte, t time.Time) ([]byte, error) {
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
func (s *BaseSigner) Verify(secret []byte, pkg *Pkg) (bool, error) {
	// convert time
	t, err := convertBytesToTime(pkg.TimeBytes)
	if err != nil {
		return false, fmt.Errorf("convert bytes to time err: %w", err)
	}
	// verify timestamp is within threshold
	if t.After(time.Now()) || t.Before(time.Now().Add(-s.sumTtl)) {
		return false, errors.New("time is outside of threshold")
	}
	// pre-allocate slice
	totalLen := len(secret) + len(pkg.TimeBytes) + len(pkg.Payload)
	data := make([]byte, 0, totalLen)
	// copy data
	data = append(data, secret...)
	data = append(data, pkg.TimeBytes...)
	data = append(data, pkg.Payload...)
	// compute checksum
	h := sha256.Sum256(data)
	// verify equality
	return bytes.Equal(pkg.Sum, h[:SumLen]), nil
}

func convertBytesToTime(b []byte) (time.Time, error) {
	if len(b) != TimeLen {
		return time.Time{}, fmt.Errorf("byte slice must be exactly 8 bytes long")
	}
	nano, err := bytesToInt64(b)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to convert bytes to int64: %w", err)
	}
	return time.Unix(0, nano), nil
}

func convertTimeToBytes(t time.Time) ([]byte, error) {
	return int64ToBytes(t.UnixNano())
}

func int64ToBytes(num int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func bytesToInt64(b []byte) (int64, error) {
	if len(b) != TimeLen {
		return 0, fmt.Errorf("byte slice must be exactly 8 bytes long")
	}

	var num int64
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.BigEndian, &num)
	if err != nil {
		return 0, err
	}
	return num, nil
}
