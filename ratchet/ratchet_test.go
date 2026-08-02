package ratchet_test

import (
	"math"
	"testing"

	"github.com/StevenACoffman/skillet/ratchet"
)

func TestEvaluate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		cand, cur, best float64
		wantAction      ratchet.Action
		wantStatus      ratchet.Status
	}{
		{"new best", 0.9, 0.5, 0.8, ratchet.AcceptNewBest, ratchet.Improved},
		{"accept not best", 0.7, 0.5, 0.8, ratchet.Accept, ratchet.Improved},
		{"reject tie", 0.5, 0.5, 0.8, ratchet.Reject, ratchet.Tie},
		{"reject regress", 0.4, 0.5, 0.8, ratchet.Reject, ratchet.Regressed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := ratchet.Evaluate(tt.cand, tt.cur, tt.best, 2, 7)
			if r.Action != tt.wantAction {
				t.Errorf("Action = %q, want %q", r.Action, tt.wantAction)
			}
			if r.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", r.Status, tt.wantStatus)
			}
		})
	}
}

func TestEvaluateNewBestStep(t *testing.T) {
	t.Parallel()
	r := ratchet.Evaluate(0.9, 0.5, 0.8, 2, 7)
	if r.BestStep != 7 {
		t.Errorf("BestStep = %d, want 7 (globalStep recorded on a new best)", r.BestStep)
	}
	if r.BestScore != 0.9 {
		t.Errorf("BestScore = %v, want 0.9", r.BestScore)
	}
}

func TestSelectScore(t *testing.T) {
	t.Parallel()
	if s, err := ratchet.SelectScore(0.8, 0.4, ratchet.Hard, 0); err != nil || s != 0.8 {
		t.Errorf("Hard = %v, %v; want 0.8", s, err)
	}
	if s, err := ratchet.SelectScore(0.8, 0.4, ratchet.Soft, 0); err != nil || s != 0.4 {
		t.Errorf("Soft = %v, %v; want 0.4", s, err)
	}
	mixed, err := ratchet.SelectScore(0.8, 0.4, ratchet.Mixed, 0.5)
	if err != nil || math.Abs(mixed-0.6) > 1e-9 {
		t.Errorf("Mixed(0.5) = %v, %v; want ~0.6", mixed, err)
	}
	if s, err := ratchet.SelectScore(0.8, 0.4, ratchet.Mixed, 2); err != nil || s != 0.4 {
		t.Errorf("Mixed weight clamp = %v, %v; want 0.4 (w clamped to 1)", s, err)
	}
	if _, err := ratchet.SelectScore(0, 0, ratchet.Metric("bogus"), 0); err == nil {
		t.Error("unknown metric should error")
	}
}

func TestActivationScore(t *testing.T) {
	t.Parallel()
	desc := "database schema migration for postgres tables"
	triggers := []string{"help me with a database migration", "alter the postgres schema"}
	decoys := []string{"what's the weather today", "database migration plan"} // 2nd overlaps
	r := ratchet.Score(desc, triggers, decoys)
	if r.Targets != 2 || r.Distractors != 2 {
		t.Fatalf("counts: targets=%d distractors=%d", r.Targets, r.Distractors)
	}
	if r.TP != 2 {
		t.Errorf("TP = %d, want 2 (both triggers share salient terms)", r.TP)
	}
	if r.FP != 1 {
		t.Errorf("FP = %d, want 1 (the 'database migration' decoy overlaps)", r.FP)
	}
	if r.TN != 1 {
		t.Errorf("TN = %d, want 1", r.TN)
	}
	if r.NetUtility != 0.25 { // (TP-FP)/total = (2-1)/4
		t.Errorf("NetUtility = %v, want 0.25", r.NetUtility)
	}
}
