package service

import (
	"sync"
	"testing"

	"pgregory.net/rapid"
)

func TestNewHealthService_DatabaseOK_DefaultsFalse(t *testing.T) {
	s := NewHealthService(nil)
	if s.DatabaseOK() {
		t.Fatal("expected DatabaseOK() to be false for a new HealthService")
	}
}

// Property 1: HealthService state reflects last ping outcome.
// For any sequence of boolean outcomes applied via setDatabaseOK,
// DatabaseOK() must equal the last value in the sequence.
func TestProperty_HealthServiceStateReflectsLastPingOutcome(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := NewHealthService(nil)
		outcomes := rapid.SliceOfN(rapid.Bool(), 1, 50).Draw(t, "outcomes")
		for _, ok := range outcomes {
			s.setDatabaseOK(ok)
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
		s := NewHealthService(nil)
		numReaders := rapid.IntRange(2, 20).Draw(t, "numReaders")
		numWrites := rapid.IntRange(1, 50).Draw(t, "numWrites")

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numWrites; i++ {
				s.setDatabaseOK(i%2 == 0)
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
