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
	if len(data) < 32+15 {
		return errors.New("data too short")
	}
	pkg.Sum = data[:32]              /*32B sha256*/
	pkg.TimeBytes = data[32 : 32+15] /*15B time*/
	pkg.Payload = data[32+15:]
	return nil
}
