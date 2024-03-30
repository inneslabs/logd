package auth

import "errors"

type Pkg struct {
	Sum,
	TimeBytes,
	Payload []byte
}

func UnpackSignedData(data []byte, pkg *Pkg) error {
	if len(data) < SumLen+TimeLen {
		return errors.New("data too short")
	}
	pkg.Sum = data[:SumLen]
	pkg.TimeBytes = data[SumLen : SumLen+TimeLen]
	pkg.Payload = data[SumLen+TimeLen:]
	return nil
}
