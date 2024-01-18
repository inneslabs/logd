/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package auth

import "errors"

type Unpacked struct {
	Sum,
	TimeBytes,
	Payload []byte
}

func UnpackSignedData(data []byte) (*Unpacked, error) {
	if len(data) < sumLen+timeLen {
		return nil, errors.New("data too short")
	}
	return &Unpacked{
		Sum:       data[:sumLen],
		TimeBytes: data[sumLen : sumLen+timeLen],
		Payload:   data[sumLen+timeLen:],
	}, nil
}
