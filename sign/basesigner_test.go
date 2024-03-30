package sign

import (
	"testing"
	"time"

	"github.com/inneslabs/logd/cmd"
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
	s := NewBaseSigner(&BaseSignerCfg{100 * time.Millisecond})
	signed, err := s.Sign(sec, payload)
	if err != nil {
		t.FailNow()
	}
	pkg := &Pkg{}
	err = UnpackSignedData(signed, pkg)
	if err != nil {
		t.FailNow()
	}
	valid, err := s.Verify(sec, pkg)
	if !valid || err != nil {
		t.Fatalf("failed with: %s", err)
	}
}

func TestSignAndVerifyInvalid(t *testing.T) {
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
	sec := []byte("testsecret")
	s := NewBaseSigner(&BaseSignerCfg{100 * time.Millisecond})
	boundary := time.Now().Add(-s.sumTtl)
	signed, err := s.signWithTime(sec, payload, boundary)
	if err != nil {
		t.FailNow()
	}
	pkg := &Pkg{}
	err = UnpackSignedData(signed, pkg)
	if err != nil {
		t.FailNow()
	}
	valid, err := s.Verify(sec, pkg)
	if valid || err == nil {
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
	s := NewBaseSigner(&BaseSignerCfg{100 * time.Millisecond})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Sign(secret, payload)
	}
}

func BenchmarkVerify(b *testing.B) {
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
	s := NewBaseSigner(&BaseSignerCfg{100 * time.Millisecond})
	signed, err := s.Sign(secret, payload)
	if err != nil {
		b.FailNow()
	}
	pkg := &Pkg{}
	err = UnpackSignedData(signed, pkg)
	if err != nil {
		b.FailNow()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Verify(secret, pkg)
	}
}
