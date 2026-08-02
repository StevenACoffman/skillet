// Package stats holds the small deterministic statistics skillet tools need. It
// is a faithful port of the pure functions in cc-thinking-skills'
// evals/lib/stats.js. Pure: values in, values out, no I/O.
package stats

import "math"

// wilsonZ is the z for a 95% two-sided Wilson score interval (matches the
// evals/lib/stats.js default of 1.959964).
const wilsonZ = 1.959964

// Wilson returns the Wilson score interval [lo, hi] for k successes out of n
// Bernoulli trials at 95% confidence. It requires 0 <= k <= n. n == 0 returns
// the maximally-uncertain [0, 1]; otherwise the bounds lie in [0, 1] and
// bracket k/n. Pure.
func Wilson(k, n int) (lo, hi float64) {
	if n == 0 {
		return 0, 1
	}
	z := wilsonZ
	fk, fn := float64(k), float64(n)
	p := fk / fn
	d := 1 + z*z/fn
	center := p + z*z/(2*fn)
	half := z * math.Sqrt(p*(1-p)/fn+z*z/(4*fn*fn))
	return (center - half) / d, (center + half) / d
}
