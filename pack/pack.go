package pack

import (
	"errors"

	"github.com/fxamacker/cbor/v2"
	"github.com/swissinfo-ch/logd/msg"
)

const (
	sumLen  = 32
	timeLen = 8
)

func UnpackSignedMsg(msg []byte) (sum, timeBytes, payload []byte, err error) {
	if len(msg) < sumLen+timeLen {
		return nil, nil, nil, errors.New("msg too short")
	}
	return msg[:sumLen],
		msg[sumLen : sumLen+timeLen],
		msg[sumLen+timeLen:],
		err
}

func PackMsg(msg *msg.Msg) ([]byte, error) {
	return cbor.Marshal(msg)
}

func UnpackMsg(data []byte) (*msg.Msg, error) {
	m := &msg.Msg{}
	err := cbor.Unmarshal(data, m)
	return m, err
}
