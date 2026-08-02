package stats_test

import (
	"math"
	"testing"

	"github.com/StevenACoffman/skillet/stats"
)

func TestWilsonEmpty(t *testing.T) {
	t.Parallel()
	lo, hi := stats.Wilson(0, 0)
	if lo != 0 || hi != 1 {
		t.Fatalf("Wilson(0,0) = [%v,%v], want [0,1]", lo, hi)
	}
}

func TestWilsonBounds(t *testing.T) {
	t.Parallel()
	cases := []struct{ k, n int }{{0, 10}, {5, 10}, {10, 10}, {1, 3}, {99, 100}}
	for _, c := range cases {
		lo, hi := stats.Wilson(c.k, c.n)
		if lo < 0 || lo > 1 || hi < 0 || hi > 1 {
			t.Errorf("Wilson(%d,%d) = [%v,%v] outside [0,1]", c.k, c.n, lo, hi)
		}
		if lo > hi {
			t.Errorf("Wilson(%d,%d): lo %v > hi %v", c.k, c.n, lo, hi)
		}
	}
}

func TestWilsonKnownInterval(t *testing.T) {
	t.Parallel()
	// 1 success in 2 trials: symmetric interval centered near 0.5.
	lo, hi := stats.Wilson(1, 2)
	if math.Abs((lo+hi)/2-0.5) > 0.05 {
		t.Errorf("Wilson(1,2) = [%v,%v]; midpoint should be near 0.5", lo, hi)
	}
}
