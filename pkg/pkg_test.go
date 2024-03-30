package pkg

import (
	"bytes"
	"testing"
)

func TestUnpack(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *Pkg
		wantErr  bool
	}{
		{
			name:    "data too short",
			data:    []byte{1, 2, 3}, // Less than 40 bytes needed
			wantErr: true,
		},
		{
			name: "data just right",
			data: bytes.Repeat([]byte{1}, 40), // Exactly 40 bytes
			expected: &Pkg{
				Sum:       bytes.Repeat([]byte{1}, 32),
				TimeBytes: bytes.Repeat([]byte{1}, 8),
				Payload:   []byte{},
			},
			wantErr: false,
		},
		{
			name: "data longer than needed",
			data: bytes.Repeat([]byte{1}, 50), // More than 40 bytes
			expected: &Pkg{
				Sum:       bytes.Repeat([]byte{1}, 32),
				TimeBytes: bytes.Repeat([]byte{1}, 8),
				Payload:   bytes.Repeat([]byte{1}, 10),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &Pkg{}
			err := Unpack(tt.data, pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unpack() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if !bytes.Equal(pkg.Sum, tt.expected.Sum) ||
					!bytes.Equal(pkg.TimeBytes, tt.expected.TimeBytes) ||
					!bytes.Equal(pkg.Payload, tt.expected.Payload) {
					t.Errorf("Unpack() got = %v, want %v", pkg, tt.expected)
				}
			}
		})
	}
}
