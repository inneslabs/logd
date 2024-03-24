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
	signed, err := Sign(sec, payload, time.Now())
	if err != nil {
		t.FailNow()
	}
	unpk := &Unpacked{}
	err = UnpackSignedData(signed, unpk)
	if err != nil {
		t.FailNow()
	}
	valid, err := Verify(sec, unpk)
	if !valid || err != nil {
		t.Fatalf("failed with: %s", err)
	}
}

func TestSignAndVerifyInvalid(t *testing.T) {
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
	signed, err := Sign(sec, payload, time.Now().Add(time.Second))
	if err != nil {
		t.FailNow()
	}
	unpk := &Unpacked{}
	err = UnpackSignedData(signed, unpk)
	if err != nil {
		t.FailNow()
	}
	valid, err := Verify(sec, unpk)
	if valid || err == nil {
		t.FailNow()
	}
}

func BenchmarkSign(b *testing.B) {
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
		b.FailNow()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sign(sec, payload, time.Now().Add(time.Second))
	}
}

func BenchmarkVerify(b *testing.B) {
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
		b.FailNow()
	}
	signed, err := Sign(sec, payload, time.Now().Add(time.Second))
	if err != nil {
		b.FailNow()
	}
	unpk := &Unpacked{}
	err = UnpackSignedData(signed, unpk)
	if err != nil {
		b.FailNow()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(sec, unpk)
	}
}
