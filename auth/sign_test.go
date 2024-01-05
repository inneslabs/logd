package auth

import (
	"testing"
	"time"

	"github.com/swissinfo-ch/logd/msg"
	"github.com/swissinfo-ch/logd/unpack"
)

func TestSignAndVerify(t *testing.T) {
	sec := []byte("testsecret")
	msg := &msg.Msg{
		Timestamp: time.Now().UnixMilli(),
		Msg:       "this is a test",
	}
	signedMsg, err := Sign(sec, msg, time.Now())
	if err != nil {
		t.FailNow()
	}
	sum, timeBytes, payload, err := unpack.UnpackMsg(signedMsg)
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
	msg := &msg.Msg{
		Timestamp: time.Now().UnixMilli(),
		Msg:       "this is a test",
	}
	signedMsg, err := Sign(sec, msg, time.Now().Add(timeThreshold*2))
	if err != nil {
		t.FailNow()
	}
	sum, timeBytes, payload, err := unpack.UnpackMsg(signedMsg)
	if err != nil {
		t.FailNow()
	}
	valid, err := Verify(sec, sum, timeBytes, payload)
	if valid || err == nil {
		t.FailNow()
	}
}
