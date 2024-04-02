package sign

import (
	"testing"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/pkg"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSignAndVerify(t *testing.T) {
	payload, err := proto.Marshal(&cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:   timestamppb.Now(),
			Txt: "this is a test",
		},
	})
	if err != nil {
		t.FailNow()
	}
	signed := Sign([]byte("testsecret"), payload)
	p := &pkg.Pkg{}
	err = pkg.Unpack(signed, p)
	if err != nil {
		t.FailNow()
	}
}

func BenchmarkSign(b *testing.B) {
	payload, err := proto.Marshal(&cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:   timestamppb.Now(),
			Txt: "test",
		},
	})
	if err != nil {
		b.FailNow()
	}
	secret := []byte("testsecret")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sign(secret, payload)
	}
}
