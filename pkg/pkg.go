package pkg

import (
	"crypto/sha256"
	"errors"
	"time"
)

type Pkg struct {
	Sum,
	TimeBytes,
	Payload []byte
}

func Unpack(data []byte, pkg *Pkg) error {
	if len(data) < 32+15 {
		return errors.New("data too short")
	}
	pkg.Sum = data[:32]              /*32B sha256*/
	pkg.TimeBytes = data[32 : 32+15] /*15B time*/
	pkg.Payload = data[32+15:]
	return nil
}

func Sign(secret, payload []byte) []byte {
	timeBytes, _ := time.Now().MarshalBinary() // 15B
	data := make([]byte, 0, 32 /* sha256 */ +15 /* time */ +len(payload))
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	h := sha256.Sum256(data)
	data = append(data[:0], h[:32]...)
	data = append(data, timeBytes...)
	return append(data, payload...)
}
