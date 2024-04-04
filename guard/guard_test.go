package guard

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/pkg"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var cfg = &Cfg{
	FilterCap: 1000000,
	FilterTtl: 10 * time.Second,
	PacketTtl: 5 * time.Minute,
}

func TestWithReplay(t *testing.T) {
	cfg := &Cfg{
		FilterCap: 1000000,
		PacketTtl: 5 * time.Minute,
	}
	guard := NewGuard(context.Background(), cfg)
	secret := []byte("secret")
	timeBytes, _ := time.Now().MarshalBinary()
	payload := []byte("payload")
	sum := calculateSum(secret, timeBytes, payload)
	p := &pkg.Pkg{
		TimeBytes: timeBytes,
		Payload:   payload,
		Sum:       sum,
	}
	require.True(t, guard.Good(secret, p), "Expected package to be good")
	require.False(t, guard.Good(secret, p), "Expected package to be rejected due to replay")
}

func TestWrongSecret(t *testing.T) {
	guard := NewGuard(context.Background(), cfg)
	timeBytes, _ := time.Now().MarshalBinary()
	payload := []byte("payload")
	sum := calculateSum([]byte("secret"), timeBytes, payload)
	require.False(t, guard.Good([]byte("wrong_secret"), &pkg.Pkg{
		TimeBytes: timeBytes,
		Payload:   payload,
		Sum:       sum,
	}))
}

func TestDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	g := NewGuard(ctx, cfg)
	go func() {
		time.After(time.Millisecond)
		cancel()
	}()
	select {
	case <-time.After(time.Second):
		t.FailNow()
	case <-g.Quit():
		fmt.Println("guard quit, looks good")
	}
}

// Re-implmemented to test
func calculateSum(secret, timeBytes, payload []byte) []byte {
	totalLen := len(secret) + len(timeBytes) + len(payload)
	data := make([]byte, 0, totalLen)
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	h := sha256.Sum256(data)
	return h[:]
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
		pkg.Sign(secret, payload)
	}
}
