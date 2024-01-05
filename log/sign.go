package log

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/swissinfo-ch/logd/msg"
)

func Sign(secret []byte, msg *msg.Msg, t time.Time) ([]byte, error) {
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(t.Unix()))
	msgPayload, err := cbor.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("cbor marshal err: %w", err)
	}
	data := append(secret, timeBytes...)
	data = append(data, msgPayload...)
	h := sha256.Sum256(data)
	return h[:32], nil
}
