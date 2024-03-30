package guard

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/inneslabs/logd/pkg"
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

	p := &pkg.Pkg{
		TimeBytes: timeBytes,
		Payload:   payload,
		Sum:       sum,
	}

	assert.True(t, guard.Good(secret, p), "Expected package to be good")

	assert.False(t, guard.Good(secret, p), "Expected package to be rejected due to replay")
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

	p := &pkg.Pkg{
		TimeBytes: timeBytes,
		Payload:   payload,
		Sum:       sum,
	}

	assert.False(t, guard.replay(sum), "Expected sum not to be found in history")

	guard.Good(secret, p) // This should add the sum to the history

	assert.True(t, guard.replay(sum), "Expected sum to be found in history due to replay")
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
