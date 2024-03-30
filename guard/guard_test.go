package guard

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"
	"time"

	"github.com/inneslabs/logd/sign"
	"github.com/stretchr/testify/assert"
)

func TestGuardGood(t *testing.T) {
	cfg := &Cfg{
		HistorySize: 10,
		SumTtl:      5 * time.Minute,
	}
	guard := NewGuard(cfg)
	secret := []byte("secret")
	currentTime := time.Now()
	timeBytes, _ := currentTime.Add(-2 * time.Minute).MarshalBinary()
	payload := []byte("payload")
	sum := calculateSum(secret, timeBytes, payload)

	pkg := &sign.Pkg{
		TimeBytes: timeBytes,
		Payload:   payload,
		Sum:       sum,
	}

	assert.True(t, guard.Good(secret, pkg), "Expected package to be good")

	assert.False(t, guard.Good(secret, pkg), "Expected package to be rejected due to replay")
}

func TestGuardReplay(t *testing.T) {
	cfg := &Cfg{
		HistorySize: 2,
		SumTtl:      5 * time.Minute,
	}
	guard := NewGuard(cfg)
	secret := []byte("secret")
	currentTime := time.Now()
	timeBytes, _ := currentTime.MarshalBinary()
	payload := []byte("payload")
	sum := calculateSum(secret, timeBytes, payload)

	pkg := &sign.Pkg{
		TimeBytes: timeBytes,
		Payload:   payload,
		Sum:       sum,
	}

	assert.False(t, guard.replay(sum), "Expected sum not to be found in history")

	guard.Good(secret, pkg) // This should add the sum to the history

	assert.True(t, guard.replay(sum), "Expected sum to be found in history due to replay")
}

func TestConvertBytesToTime(t *testing.T) {
	currentTime := time.Now()
	timeBytes, _ := currentTime.MarshalBinary()

	convertedTime, err := convertBytesToTime(timeBytes)
	assert.NoError(t, err, "Expected no error converting bytes to time")
	assert.Equal(t, currentTime.Round(time.Second), convertedTime.Round(time.Second), "Expected times to be equal")
}

func TestBytesToInt64(t *testing.T) {
	expected := int64(1234567890)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(expected))

	result, err := bytesToInt64(b)
	assert.NoError(t, err, "Expected no error converting bytes to int64")
	assert.Equal(t, expected, result, "Expected int64 values to be equal")
}

// Helper function to calculate sum for testing
func calculateSum(secret, timeBytes, payload []byte) []byte {
	totalLen := len(secret) + len(timeBytes) + len(payload)
	data := make([]byte, 0, totalLen)
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	h := sha256.Sum256(data)
	return h[:]
}
