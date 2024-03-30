package auth

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
	signed, err := SignWithTime(sec, payload, time.Now())
	if err != nil {
		t.FailNow()
	}
	pkg := &Pkg{}
	err = UnpackSignedData(signed, pkg)
	if err != nil {
		t.FailNow()
	}
	valid, err := Verify(sec, pkg)
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
	boundary := time.Now().Add(-SigTtl)
	signed, err := SignWithTime(sec, payload, boundary)
	if err != nil {
		t.FailNow()
	}
	pkg := &Pkg{}
	err = UnpackSignedData(signed, pkg)
	if err != nil {
		t.FailNow()
	}
	valid, err := Verify(sec, pkg)
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sign(secret, payload)
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
	signed, err := Sign(secret, payload)
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
		Verify(secret, pkg)
	}
}
