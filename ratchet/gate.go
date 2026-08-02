// Package ratchet is the acceptance machinery of a darwin-style optimization
// loop: the keep/revert gate (gate.go) and the activation confusion matrix that
// feeds it a trigger-accuracy signal (activation.go). Both are pure — values in,
// values out — a direct port of SkillOpt's decision functions.
package ratchet

import "fmt"

// Metric value constants select how a (hard, soft) pair is projected onto one
// comparable score.
const (
	Hard  Metric = "hard"
	Soft  Metric = "soft"
	Mixed Metric = "mixed"
)

// Action value constants are the possible gate outcomes (the disposition axis).
const (
	AcceptNewBest Action = "accept_new_best"
	Accept        Action = "accept"
	Reject        Action = "reject"
)

// Status value constants are the measured direction of the candidate vs the
// current score (the measured axis, kept separate from the disposition Action:
// a candidate can be Improved yet still not the new best).
const (
	Improved  Status = "improved"
	Tie       Status = "tie"
	Regressed Status = "regressed"
)

// Metric selects how a (hard, soft) pair is projected onto one comparable score.
type Metric string

// Action is the gate outcome (disposition axis).
type Action string

// Status is the measured direction of the candidate vs the current score.
type Status string

// Result is the immutable gate outcome plus the resulting state. Action is the
// disposition (keep/revert/new-best); Delta and Status are the separate measured
// axis (how the candidate moved).
type Result struct {
	Action       Action  `json:"action"`
	Status       Status  `json:"status"`
	Delta        float64 `json:"delta"` // candidate - current
	CurrentScore float64 `json:"current_score"`
	BestScore    float64 `json:"best_score"`
	BestStep     int     `json:"best_step"`
}

// SelectScore projects (hard, soft) onto a single metric. Callers usually pass
// one weighted total as "hard" with the default Hard metric.
func SelectScore(hard, soft float64, metric Metric, mixedWeight float64) (float64, error) {
	switch metric {
	case Hard:
		return hard, nil
	case Soft:
		return soft, nil
	case Mixed:
		w := mixedWeight
		if w < 0 {
			w = 0
		}
		if w > 1 {
			w = 1
		}
		return (1.0-w)*hard + w*soft, nil
	default:
		return 0, fmt.Errorf("ratchet: unknown metric %q; expected hard, soft, or mixed", metric)
	}
}

// Evaluate compares an already-projected candidate score against current/best
// using strict ">" at both comparisons. globalStep is recorded as the new best
// step when a new best is accepted. Ties reject and do not promote.
func Evaluate(cand, current, best float64, bestStep, globalStep int) Result {
	var r Result
	switch {
	case cand > current && cand > best:
		r = Result{Action: AcceptNewBest, CurrentScore: cand, BestScore: cand, BestStep: globalStep}
	case cand > current:
		r = Result{Action: Accept, CurrentScore: cand, BestScore: best, BestStep: bestStep}
	default:
		r = Result{Action: Reject, CurrentScore: current, BestScore: best, BestStep: bestStep}
	}
	r.Delta = cand - current
	r.Status = statusOf(r.Delta)
	return r
}

// statusOf maps a candidate-minus-current delta onto the measured direction.
func statusOf(delta float64) Status {
	switch {
	case delta > 0:
		return Improved
	case delta < 0:
		return Regressed
	default:
		return Tie
	}
}
