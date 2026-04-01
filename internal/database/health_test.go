package database

import (
	"sync"
	"testing"

	"pgregory.net/rapid"
)

func TestNewHealthChecker_DatabaseOK_DefaultsFalse(t *testing.T) {
	s := NewHealthChecker(nil)
	if s.DatabaseOK() {
		t.Fatal("expected DatabaseOK() to be false for a new HealthChecker")
	}
}

// Property 1: HealthChecker state reflects last ping outcome.
// For any sequence of boolean outcomes applied via updateDBStatus,
// DatabaseOK() must equal the last value in the sequence.
func TestProperty_HealthCheckerStateReflectsLastPingOutcome(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := NewHealthChecker(nil)
		outcomes := rapid.SliceOfN(rapid.Bool(), 1, 50).Draw(t, "outcomes")
		for _, ok := range outcomes {
			s.updateDBStatus(ok)
		}
		last := outcomes[len(outcomes)-1]
		if s.DatabaseOK() != last {
			t.Fatalf("expected DatabaseOK()=%v, got %v", last, s.DatabaseOK())
		}
	})
}

// Property 3: Concurrent reads are data-race free.
// For any number of concurrent readers and a writer, all reads must complete
// without data races or panics.
func TestProperty_ConcurrentReadsDataRaceFree(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := NewHealthChecker(nil)
		numReaders := rapid.IntRange(2, 20).Draw(t, "numReaders")
		numWrites := rapid.IntRange(1, 50).Draw(t, "numWrites")

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numWrites; i++ {
				s.updateDBStatus(i%2 == 0)
			}
		}()

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = s.DatabaseOK()
			}()
		}

		wg.Wait()
	})
}
