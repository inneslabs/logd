/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package auth

import "errors"

func UnpackSignedMsg(msg []byte) (sum, timeBytes, payload []byte, err error) {
	if len(msg) < sumLen+timeLen {
		return nil, nil, nil, errors.New("msg too short")
	}
	return msg[:sumLen],
		msg[sumLen : sumLen+timeLen],
		msg[sumLen+timeLen:],
		err
}
