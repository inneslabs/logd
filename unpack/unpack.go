package unpack

import (
	"errors"
)

const (
	sumLen  = 32
	timeLen = 8
)

func UnpackMsg(msg []byte) (sum, timeBytes, payload []byte, err error) {
	if len(msg) < sumLen+timeLen {
		return nil, nil, nil, errors.New("msg too short")
	}
	return msg[:sumLen],
		msg[sumLen : sumLen+timeLen],
		msg[sumLen+timeLen:],
		err
}
