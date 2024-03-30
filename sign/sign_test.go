package sign

import (
	"testing"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/guard"
	"github.com/inneslabs/logd/pkg"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSignAndVerify(t *testing.T) {
	sec := []byte("testsecret")
	txt := "this is a test"
	payload, err := proto.Marshal(&cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:   timestamppb.Now(),
			Txt: &txt,
		},
	})
	if err != nil {
		t.FailNow()
	}
	signed := Sign(sec, payload)
	p := &pkg.Pkg{}
	err = pkg.Unpack(signed, p)
	if err != nil {
		t.FailNow()
	}
	g := guard.NewGuard(&guard.Cfg{
		HistorySize: 0,
		SumTtl:      100 * time.Millisecond,
	})
	if !g.Good(sec, p) {
		t.FailNow()
	}
}

func BenchmarkSign(b *testing.B) {
	txt := "test"
	payload, err := proto.Marshal(&cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:   timestamppb.Now(),
			Txt: &txt,
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
