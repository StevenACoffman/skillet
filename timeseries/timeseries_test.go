package timeseries_test

import (
	"math"
	"testing"

	"github.com/StevenACoffman/skillet/timeseries"
)

func TestDetectWithdrawsFromJudgingWhenHistoryIsTooShort(t *testing.T) {
	t.Parallel()
	// A gate that fails the first time a metric is recorded is worse than no gate:
	// the only way to make it pass is to stop measuring.
	cases := map[string][]float64{
		"no history at all": {},
		"a single run":      {9.0},
	}
	for name, history := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := timeseries.Detect(history, 0.0, timeseries.Config{})
			if v.Compared {
				t.Errorf("compared against %d measurements, want no verdict", len(history))
			}
			if v.Regressed {
				t.Error("reported a regression with no baseline to regress from")
			}
			if v.Current != 0.0 {
				t.Errorf("Current = %v, want the measurement echoed back", v.Current)
			}
		})
	}
}

func TestDetectMinHistoryIsConfigurable(t *testing.T) {
	t.Parallel()
	history := []float64{8, 8, 8}
	if v := timeseries.Detect(history, 1, timeseries.Config{MinHistory: 4}); v.Compared {
		t.Error("compared with fewer measurements than MinHistory")
	}
	if v := timeseries.Detect(history, 1, timeseries.Config{MinHistory: 3}); !v.Compared {
		t.Error("refused to compare with exactly MinHistory measurements")
	}
}

func TestBaselineIsTheWindowMeanNotTheLastValue(t *testing.T) {
	t.Parallel()
	// The correction that matters most: with the last value as the baseline, one
	// unusually good run becomes the bar for every run after it. Averaging does not
	// erase that run's influence — it dilutes it to 1/N, which is the whole point.
	// Here the last run spiked to 10 while the norm is 6.
	history := []float64{6, 6, 6, 10}
	v := timeseries.Detect(history, 6, timeseries.Config{Tolerance: 1.5})
	if !v.Compared {
		t.Fatal("expected a comparison")
	}
	if math.Abs(v.Baseline-7.0) > 1e-9 {
		t.Errorf("Baseline = %v, want the mean 7.0 (the last value alone would be 10)", v.Baseline)
	}
	if math.Abs(v.Drop-1.0) > 1e-9 {
		t.Errorf("Drop = %v, want 1.0; against the last value alone it would be 4.0", v.Drop)
	}
	// The same tolerance applied to a last-value baseline would have convicted this
	// perfectly normal measurement.
	if v.Regressed {
		t.Errorf("a normal measurement one spike later must not regress: %+v", v)
	}
	if lastValueDrop := 10.0 - 6.0; lastValueDrop <= 1.5 {
		t.Fatal("test is not exercising the difference; pick a larger spike")
	}
}

func TestWindowLimitsTheBaselineToRecentMeasurements(t *testing.T) {
	t.Parallel()
	// Old history that no longer reflects the system must not anchor the baseline.
	history := []float64{1, 1, 1, 1, 9, 9}
	v := timeseries.Detect(history, 9, timeseries.Config{Window: 2})
	if v.N != 2 {
		t.Errorf("N = %d, want the 2 windowed measurements", v.N)
	}
	if math.Abs(v.Baseline-9.0) > 1e-9 {
		t.Errorf("Baseline = %v, want 9.0 from the window alone", v.Baseline)
	}
	all := timeseries.Detect(history, 9, timeseries.Config{})
	if all.N != len(history) {
		t.Errorf("an unset Window must use all %d measurements, used %d", len(history), all.N)
	}
}

func TestAWindowLargerThanHistoryUsesWhatExists(t *testing.T) {
	t.Parallel()
	v := timeseries.Detect([]float64{4, 6}, 5, timeseries.Config{Window: 100})
	if v.N != 2 || math.Abs(v.Baseline-5.0) > 1e-9 {
		t.Errorf("got N=%d baseline=%v, want N=2 baseline=5", v.N, v.Baseline)
	}
}

func TestAZeroBaselineIsStillABaseline(t *testing.T) {
	t.Parallel()
	// The reference implementation treats baseline==0 as "no baseline", so a metric
	// sitting at zero can never report a regression — precisely when a gate matters.
	// Here the metric was 0 and went negative, which is a real drop.
	v := timeseries.Detect([]float64{0, 0, 0}, -1, timeseries.Config{})
	if !v.Compared {
		t.Fatal("a zero baseline was mistaken for an absent one")
	}
	if !v.Regressed {
		t.Errorf("a drop below a zero baseline must regress: %+v", v)
	}
}

func TestToleranceIsAbsoluteAndInclusive(t *testing.T) {
	t.Parallel()
	history := []float64{8, 8, 8} // baseline 8
	cases := map[string]struct {
		current   float64
		tolerance float64
		want      bool
	}{
		"drop smaller than tolerance":    {current: 7.8, tolerance: 0.5, want: false},
		"drop exactly at tolerance":      {current: 7.5, tolerance: 0.5, want: false},
		"drop past tolerance":            {current: 7.4, tolerance: 0.5, want: true},
		"any drop when tolerance is nil": {current: 7.999, tolerance: 0, want: true},
		"no drop at all":                 {current: 8, tolerance: 0, want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := timeseries.Detect(history, tc.current, timeseries.Config{Tolerance: tc.tolerance})
			if v.Regressed != tc.want {
				t.Errorf("Regressed = %t, want %t (%+v)", v.Regressed, tc.want, v)
			}
		})
	}
}

func TestAnImprovementReportsANegativeDrop(t *testing.T) {
	t.Parallel()
	v := timeseries.Detect([]float64{5, 5}, 8, timeseries.Config{})
	if v.Regressed {
		t.Error("an improvement was reported as a regression")
	}
	if v.Drop >= 0 {
		t.Errorf("Drop = %v, want negative for an improvement", v.Drop)
	}
}

func TestDetectDoesNotMutateHistory(t *testing.T) {
	t.Parallel()
	history := []float64{1, 2, 3, 4}
	before := append([]float64(nil), history...)
	timeseries.Detect(history, 0, timeseries.Config{Window: 2})
	for i := range history {
		if history[i] != before[i] {
			t.Fatalf("Detect mutated history: %v, was %v", history, before)
		}
	}
}
