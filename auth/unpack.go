/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package auth

import "errors"

func UnpackSignedData(data []byte) (sum, timeBytes, payload []byte, err error) {
	if len(data) < sumLen+timeLen {
		return nil, nil, nil, errors.New("data too short")
	}
	return data[:sumLen],
		data[sumLen : sumLen+timeLen],
		data[sumLen+timeLen:],
		err
}
