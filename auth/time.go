package auth

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

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
