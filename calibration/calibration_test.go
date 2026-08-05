package calibration_test

import (
	"testing"

	"github.com/StevenACoffman/skillet/calibration"
)

const tol = 1e-9

func approx(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= tol
}

// TestPerfectlyCalibratedIrreducibleBrier: four 0.5 predictions, half correct, is
// perfectly calibrated (accuracy == confidence in its bin, so ECE and MCE are 0),
// yet still carries the irreducible Brier of a hedged guess (0.25).
func TestPerfectlyCalibratedIrreducibleBrier(t *testing.T) {
	t.Parallel()
	rep := calibration.Compute([]calibration.Sample{
		{Confidence: 0.5, Correct: true},
		{Confidence: 0.5, Correct: false},
		{Confidence: 0.5, Correct: true},
		{Confidence: 0.5, Correct: false},
	})
	if !approx(rep.ECE, 0) || !approx(rep.MCE, 0) {
		t.Errorf("ECE=%v MCE=%v, want both 0", rep.ECE, rep.MCE)
	}
	if !approx(rep.Brier, 0.25) {
		t.Errorf("Brier=%v, want 0.25", rep.Brier)
	}
	if rep.Samples != 4 {
		t.Errorf("Samples=%d, want 4", rep.Samples)
	}
	if len(rep.Buckets) != 1 || rep.Buckets[0].Count != 4 {
		t.Fatalf("Buckets=%+v, want one bin of 4", rep.Buckets)
	}
}

// TestOverconfident: ten 0.9 predictions all wrong — maximally miscalibrated in a
// single bin, so ECE == MCE == 0.9 and Brier == 0.81.
func TestOverconfident(t *testing.T) {
	t.Parallel()
	samples := make([]calibration.Sample, 10)
	for i := range samples {
		samples[i] = calibration.Sample{Confidence: 0.9, Correct: false}
	}
	rep := calibration.Compute(samples)
	if !approx(rep.ECE, 0.9) || !approx(rep.MCE, 0.9) {
		t.Errorf("ECE=%v MCE=%v, want both 0.9", rep.ECE, rep.MCE)
	}
	if !approx(rep.Brier, 0.81) {
		t.Errorf("Brier=%v, want 0.81", rep.Brier)
	}
}

// TestEmptyIsZeroReport: no samples yields the zero Report, no panic.
func TestEmptyIsZeroReport(t *testing.T) {
	t.Parallel()
	rep := calibration.Compute(nil)
	if rep.Samples != 0 || rep.ECE != 0 || rep.MCE != 0 || rep.Brier != 0 || rep.Buckets != nil {
		t.Errorf("empty Compute = %+v, want zero Report", rep)
	}
}

// TestOutOfRangeSkipped: a confidence outside [0,1] is excluded from every metric
// (this is the rigor fix over the reference, which divided by the raw length).
func TestOutOfRangeSkipped(t *testing.T) {
	t.Parallel()
	rep := calibration.Compute([]calibration.Sample{
		{Confidence: 1.5, Correct: true},   // skipped
		{Confidence: -0.2, Correct: false}, // skipped
		{Confidence: 0.9, Correct: false},
	})
	if rep.Samples != 1 {
		t.Fatalf("Samples=%d, want 1 (two skipped)", rep.Samples)
	}
	if !approx(rep.Brier, 0.81) { // (0.9-0)^2 over 1 scored sample, not 3
		t.Errorf("Brier=%v, want 0.81 (divided by scored, not input length)", rep.Brier)
	}
}

// TestInvariants sweeps constructed sample sets and asserts the properties that
// must hold for any input: every metric is bounded in [0,1] and MCE ≥ ECE (a
// weighted mean of per-bin gaps cannot exceed their maximum). Deterministic — the
// inputs vary by index, no randomness.
func TestInvariants(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 50; n++ {
		samples := make([]calibration.Sample, n)
		for i := range samples {
			samples[i] = calibration.Sample{
				Confidence: float64((i*7+n)%11) / 10.0, // 0.0..1.0 across bins
				Correct:    (i*3+n)%2 == 0,
			}
		}
		rep := calibration.Compute(samples)
		if rep.ECE < 0 || rep.ECE > 1 || rep.MCE < 0 || rep.MCE > 1 || rep.Brier < 0 ||
			rep.Brier > 1 {
			t.Fatalf("n=%d: metric out of [0,1]: %+v", n, rep)
		}
		if rep.MCE+tol < rep.ECE {
			t.Fatalf("n=%d: MCE %v < ECE %v", n, rep.MCE, rep.ECE)
		}
	}
}

// TestPerfectCalibrationZeroECE: when each bin's accuracy exactly matches its
// confidence, ECE and MCE are zero regardless of Brier.
func TestPerfectCalibrationZeroECE(t *testing.T) {
	t.Parallel()
	// Bin at 0.0: 3 wrong (accuracy 0.0). Bin at 1.0: 3 right (accuracy 1.0).
	rep := calibration.Compute([]calibration.Sample{
		{Confidence: 0, Correct: false},
		{Confidence: 0, Correct: false},
		{Confidence: 0, Correct: false},
		{Confidence: 1, Correct: true},
		{Confidence: 1, Correct: true},
		{Confidence: 1, Correct: true},
	})
	if !approx(rep.ECE, 0) || !approx(rep.MCE, 0) || !approx(rep.Brier, 0) {
		t.Errorf("perfect calibration = %+v, want ECE/MCE/Brier all 0", rep)
	}
}
