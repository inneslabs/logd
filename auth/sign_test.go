package auth

import (
	"testing"
	"time"

	"github.com/swissinfo-ch/logd/msg"
	"github.com/swissinfo-ch/logd/pack"
)

func TestSignAndVerify(t *testing.T) {
	sec := []byte("testsecret")
	payload, err := pack.PackMsg(&msg.Msg{
		Timestamp: time.Now().UnixMilli(),
		Msg:       "this is a test",
	})
	if err != nil {
		t.FailNow()
	}
	signedMsg, err := Sign(sec, payload, time.Now())
	if err != nil {
		t.FailNow()
	}
	sum, timeBytes, payload, err := pack.UnpackSignedMsg(signedMsg)
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
	payload, err := pack.PackMsg(&msg.Msg{
		Timestamp: time.Now().UnixMilli(),
		Msg:       "this is a test",
	})
	if err != nil {
		t.FailNow()
	}
	signedMsg, err := Sign(sec, payload, time.Now().Add(time.Second))
	if err != nil {
		t.FailNow()
	}
	sum, timeBytes, payload, err := pack.UnpackSignedMsg(signedMsg)
	if err != nil {
		t.FailNow()
	}
	valid, err := Verify(sec, sum, timeBytes, payload)
	if valid || err == nil {
		t.FailNow()
	}
}
