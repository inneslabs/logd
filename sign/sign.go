package sign

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

type Pkg struct {
	Sum,
	TimeBytes,
	Payload []byte
}

func UnpackSignedData(data []byte, pkg *Pkg) error {
	if len(data) < 32+8 {
		return errors.New("data too short")
	}
	pkg.Sum = data[:32]
	pkg.TimeBytes = data[32 : 32+8]
	pkg.Payload = data[32+8:]
	return nil
}

func Sign(secret, payload []byte) ([]byte, error) {
	timeBytes, err := convertTimeToBytes(time.Now())
	if err != nil {
		return nil, fmt.Errorf("convert time to bytes err: %w", err)
	}
	// pre-allocate slice
	totalLen := 32 + 8 + len(payload)
	data := make([]byte, 0, totalLen)
	// copy data
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	// compute checksum
	h := sha256.Sum256(data)
	// append sum and timeBytes to data slice
	data = append(data[:0], h[:32]...)
	data = append(data, timeBytes...)
	return append(data, payload...), nil
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
