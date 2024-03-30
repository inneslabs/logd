package pkg

import (
	"errors"
)

type Pkg struct {
	Sum,
	TimeBytes,
	Payload []byte
}

func Unpack(data []byte, pkg *Pkg) error {
	if len(data) < 32+8 {
		return errors.New("data too short")
	}
	pkg.Sum = data[:32]
	pkg.TimeBytes = data[32 : 32+8]
	pkg.Payload = data[32+8:]
	return nil
}
