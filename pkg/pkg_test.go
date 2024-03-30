package pkg

import (
	"reflect"
	"testing"
)

func TestUnpackValidData(t *testing.T) {
	data := append(append(make([]byte, 32), make([]byte, 15)...), "payload data"...)
	expectedPkg := Pkg{
		Sum:       make([]byte, 32),
		TimeBytes: make([]byte, 15),
		Payload:   []byte("payload data"),
	}

	var pkg Pkg
	err := Unpack(data, &pkg)
	if err != nil {
		t.Errorf("Unpack() error = %v, wantErr %v", err, false)
	}
	if !reflect.DeepEqual(pkg, expectedPkg) {
		t.Errorf("Unpack() = %v, want %v", pkg, expectedPkg)
	}
}

func TestUnpackDataTooShort(t *testing.T) {
	data := []byte("short data")
	var pkg Pkg
	err := Unpack(data, &pkg)
	if err == nil {
		t.Errorf("Unpack() error = %v, wantErr %v", err, true)
	}
}
