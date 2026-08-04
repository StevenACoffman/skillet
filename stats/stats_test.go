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

func TestMcNemarSignificance(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                string
		improved, regressed int
		wantSig             bool
	}{
		{"no discordant pairs", 0, 0, false},
		{"three discordant pairs below the critical value", 2, 1, false},
		{"twelve one-directional flips is significant", 12, 0, true},
		{"balanced flips are never significant", 20, 20, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if _, sig := stats.McNemar(c.improved, c.regressed); sig != c.wantSig {
				t.Errorf("McNemar(%d,%d) significant = %v, want %v",
					c.improved, c.regressed, sig, c.wantSig)
			}
		})
	}
}

func TestMcNemarStatistic(t *testing.T) {
	t.Parallel()
	// No discordant pairs → zero statistic. 12 vs 0: corrected (|12-0|-1)^2/12 = 121/12.
	if stat, _ := stats.McNemar(0, 0); stat != 0 {
		t.Errorf("McNemar(0,0) stat = %v, want 0", stat)
	}
	if stat, _ := stats.McNemar(12, 0); math.Abs(stat-121.0/12.0) > 1e-9 {
		t.Errorf("McNemar(12,0) stat = %v, want %v", stat, 121.0/12.0)
	}
}
