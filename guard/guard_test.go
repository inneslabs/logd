package guard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGuard(t *testing.T) {
	cfg := &Cfg{HistorySize: 3}
	guard := NewGuard(cfg)

	assert.NotNil(t, guard)
	assert.Equal(t, 3, guard.history.Len())
}

func TestGuard_Good_NewEntry(t *testing.T) {
	cfg := &Cfg{HistorySize: 3}
	guard := NewGuard(cfg)

	sum := []byte("new entry")
	isGood := guard.Good(sum)

	assert.True(t, isGood, "Expected new entry to be considered good")
}

func TestGuard_Good_DuplicateEntry(t *testing.T) {
	cfg := &Cfg{HistorySize: 3}
	guard := NewGuard(cfg)

	sum1 := []byte("entry")
	guard.Good(sum1) // Add first entry

	isGoodDuplicate := guard.Good(sum1) // Try to add it again

	assert.False(t, isGoodDuplicate, "Expected duplicate entry to be considered not good")
}

func TestGuard_Good_OverwriteOldestWhenFull(t *testing.T) {
	cfg := &Cfg{HistorySize: 2}
	guard := NewGuard(cfg)

	sum1 := []byte("first")
	sum2 := []byte("second")
	sum3 := []byte("third")

	guard.Good(sum1)
	guard.Good(sum2)

	// At this point, the history is full. Adding a third entry should overwrite the first one.
	guard.Good(sum3)

	// Verify that the third entry is considered good and the first one can be added again as good.
	assert.True(t, guard.Good(sum1), "Expected the first entry to be considered good after history is overwritten")
}
