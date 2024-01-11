/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package auth

import (
	"testing"
	"time"

	"github.com/swissinfo-ch/logd/cmd"
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
	sum, timeBytes, payload, err := UnpackSignedData(signed)
	if err != nil {
		t.FailNow()
	}
	valid, err := Verify(sec, sum, timeBytes, payload)
	if !valid || err != nil {
		t.FailNow()
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
	sum, timeBytes, payload, err := UnpackSignedData(signed)
	if err != nil {
		t.FailNow()
	}
	valid, err := Verify(sec, sum, timeBytes, payload)
	if valid || err == nil {
		t.FailNow()
	}
}
