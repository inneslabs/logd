package guard

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/pkg"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestSignAndVerify(t *testing.T) {
	payload, err := proto.Marshal(&cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:   timestamppb.Now(),
			Txt: "this is a test",
		},
	})
	if err != nil {
		t.FailNow()
	}
	signed := Sign([]byte("testsecret"), payload)
	p := &pkg.Pkg{}
	err = pkg.Unpack(signed, p)
	if err != nil {
		t.FailNow()
	}
}

func BenchmarkSign(b *testing.B) {
	payload, err := proto.Marshal(&cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:   timestamppb.Now(),
			Txt: "test",
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
