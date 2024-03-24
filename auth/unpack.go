package auth

import "errors"

type Unpacked struct {
	Sum,
	TimeBytes,
	Payload []byte
}

func UnpackSignedData(data []byte, unpk *Unpacked) error {
	if len(data) < SumLen+TimeLen {
		return errors.New("data too short")
	}
	unpk.Sum = data[:SumLen]
	unpk.TimeBytes = data[SumLen : SumLen+TimeLen]
	unpk.Payload = data[SumLen+TimeLen:]
	return nil
}
