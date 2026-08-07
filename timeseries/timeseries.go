// Package timeseries detects a quality regression: whether a new measurement has
// fallen below the baseline set by the ones before it.
//
// This asks a different question from a fixed threshold. A threshold asks "is this good
// enough?"; a regression gate asks "is this worse than we were?" — which catches a slow
// slide that never crosses the absolute bar, and tolerates a metric that is legitimately
// low but stable. A gate wants both, for different reasons.
//
// Storage is the caller's concern: Detect takes the history as values, so a series kept
// in a JSON file, a database, or a CI artifact all use the same code. Every function
// here is pure.
package timeseries

// defaultMinHistory is the fewest prior measurements that may form a baseline when
// Config leaves MinHistory unset.
//
// Two, not one: a single prior run makes that run the bar, so one unusually good result
// convicts every normal one after it. Requiring two is the least that averages anything.
const defaultMinHistory = 2

// Config tunes the comparison. The zero value is usable: it takes the whole history as
// the baseline window, requires defaultMinHistory measurements, and treats any drop at
// all as a regression.
type Config struct {
	// Window is how many of the most recent measurements form the baseline. Zero or
	// negative means all of them. A window keeps the baseline tracking recent
	// behaviour rather than being anchored by history that no longer reflects the
	// system.
	Window int

	// MinHistory is the fewest measurements that may form a baseline. Below it Detect
	// reports Compared false and never a regression: a gate that fails the first time
	// a metric is recorded is worse than no gate, because the only way to fix it is to
	// stop measuring.
	MinHistory int

	// Tolerance is how far below the baseline is still acceptable, in the metric's own
	// units. It is deliberately absolute rather than a fraction of the baseline: a
	// relative drop is undefined at a baseline of zero and explodes near it, and the
	// metrics in this family (a 1-10 rubric score, a 0-1 accuracy) have meaningful
	// absolute scales. A zero Tolerance means any drop is a regression, which is
	// usually too strict for a noisy metric — set it deliberately.
	Tolerance float64
}

// Verdict is the outcome of one comparison.
type Verdict struct {
	// Compared is false when there was too little history to form a baseline. It is
	// distinct from Regressed being false: "we have no opinion" and "this is fine" are
	// different answers, and a caller that conflates them silently passes every early
	// run. The remaining fields other than Current are meaningless when it is false.
	Compared bool

	// Regressed is true when the drop below the baseline exceeded the tolerance.
	Regressed bool

	Current  float64 // the measurement being judged
	Baseline float64 // mean of the window
	N        int     // measurements that went into the baseline
	Drop     float64 // Baseline - Current; negative when the measurement improved
}

// Detect judges current against the baseline formed by history.
//
// history is in chronological order, oldest first, and does not include current. The
// baseline is the mean of the last Config.Window measurements — a mean, not the single
// most recent value, so one anomalous run neither becomes the bar for everything after
// it nor is mistaken for a trend.
//
// A baseline of zero is a real baseline. Treating it as "no baseline available" would
// mean a metric genuinely sitting at zero could never report a regression, which is
// exactly when a gate matters most; absence of history is carried by Compared instead.
//
// Ensures: it is pure and does not mutate or retain history; Regressed is false
//
//	whenever Compared is false.
func Detect(history []float64, current float64, cfg Config) Verdict {
	minHistory := cfg.MinHistory
	if minHistory <= 0 {
		minHistory = defaultMinHistory
	}
	if len(history) < minHistory {
		return Verdict{Current: current}
	}
	window := history
	if cfg.Window > 0 && len(window) > cfg.Window {
		window = window[len(window)-cfg.Window:]
	}
	var sum float64
	for _, v := range window {
		sum += v
	}
	baseline := sum / float64(len(window))
	drop := baseline - current
	return Verdict{
		Compared:  true,
		Regressed: drop > cfg.Tolerance,
		Current:   current,
		Baseline:  baseline,
		N:         len(window),
		Drop:      drop,
	}
}
