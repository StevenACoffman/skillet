// Package calibration measures how well stated confidences match observed
// outcomes — the reliability complement to package stats, which owns
// significance. A well-calibrated confidence of 0.8 is correct about 80% of the
// time. Pure: values in, values out, no I/O.
//
// The formulas — Expected and Maximum Calibration Error and the Brier score, over
// equal-width confidence bins comparing each bin's accuracy to its mean stated
// confidence — are ported from unified-thinking's
// benchmarks/evaluators/calibration.go, with one fix: every metric divides by the
// number of in-range samples actually scored, not the raw input length, so an
// out-of-range sample cannot skew the result.
package calibration

import "math"

// buckets is the number of equal-width confidence bins partitioning [0,1].
const buckets = 10

// Sample is one recorded prediction: the confidence stated when it was made and
// whether it turned out correct.
type Sample struct {
	Confidence float64 // stated confidence, in [0,1]
	Correct    bool    // whether the prediction was right
}

// Bucket is one confidence bin's calibration: how many samples fell in it, the
// mean confidence stated within it, and the fraction that were correct. A
// well-calibrated bin has Accuracy close to Confidence.
type Bucket struct {
	Count      int     // samples in this bin
	Confidence float64 // mean stated confidence within the bin
	Accuracy   float64 // fraction correct within the bin
}

// Report is the calibration of a sample set: the three summary errors and the
// per-bin breakdown they derive from. Lower ECE, MCE, and Brier are better; each
// lies in [0,1], and MCE is always at least ECE.
type Report struct {
	ECE     float64  // Expected Calibration Error: sample-weighted mean |accuracy − confidence|
	MCE     float64  // Maximum Calibration Error: the worst bin's |accuracy − confidence|
	Brier   float64  // Brier score: mean squared (confidence − outcome)
	Samples int      // in-range samples scored (out-of-range confidences are skipped)
	Buckets []Bucket // non-empty bins, ordered low confidence to high
}

// Compute returns the calibration Report for samples. A sample whose Confidence
// falls outside [0,1] is skipped and excluded from every metric. Empty or
// all-skipped input yields the zero Report. Pure.
func Compute(samples []Sample) Report {
	type bin struct {
		count   int
		correct int
		sumConf float64
	}
	bins := make([]bin, buckets)
	var scored int
	var brierSum float64
	for _, s := range samples {
		if s.Confidence < 0 || s.Confidence > 1 {
			continue
		}
		idx := int(s.Confidence * buckets)
		if idx >= buckets {
			idx = buckets - 1 // confidence == 1.0 lands in the top bin
		}
		outcome := 0.0
		if s.Correct {
			bins[idx].correct++
			outcome = 1.0
		}
		bins[idx].count++
		bins[idx].sumConf += s.Confidence
		diff := s.Confidence - outcome
		brierSum += diff * diff
		scored++
	}
	if scored == 0 {
		return Report{}
	}
	rep := Report{Brier: brierSum / float64(scored), Samples: scored}
	for _, b := range bins {
		if b.count == 0 {
			continue
		}
		accuracy := float64(b.correct) / float64(b.count)
		avgConf := b.sumConf / float64(b.count)
		gap := math.Abs(accuracy - avgConf)
		rep.ECE += float64(b.count) / float64(scored) * gap
		if gap > rep.MCE {
			rep.MCE = gap
		}
		rep.Buckets = append(
			rep.Buckets,
			Bucket{Count: b.count, Confidence: avgConf, Accuracy: accuracy},
		)
	}
	return rep
}
