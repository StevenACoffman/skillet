package identity_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/identity"
)

func TestHashKnownVectors(t *testing.T) {
	t.Parallel()
	// The first 16 hex chars of the canonical SHA-256 test vectors. Freezing
	// these guards the cross-tool identity contract against accidental change.
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"empty", "", "e3b0c44298fc1c14"},
		{"abc", "abc", "ba7816bf8f01cfea"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := identity.Hash(tt.content); got != tt.want {
				t.Fatalf("Hash(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestHashProperties(t *testing.T) {
	t.Parallel()

	got := identity.Hash("anything")
	if len(got) != 16 {
		t.Fatalf("Hash length = %d, want 16", len(got))
	}
	if strings.TrimLeft(got, "0123456789abcdef") != "" {
		t.Fatalf("Hash(...) = %q, want only lowercase hex chars", got)
	}
	first := identity.Hash("x")
	if again := identity.Hash("x"); first != again {
		t.Fatalf("Hash not deterministic: %q then %q", first, again)
	}
	if identity.Hash("a") == identity.Hash("b") {
		t.Fatal("Hash collided on distinct inputs")
	}
}
