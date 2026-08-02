// Package identity provides content-identity hashing shared across skillet
// tools. The hash is byte-identical to SkillOpt's skill_hash and to the
// skill.Hash used by exegesis and skillsaw, so a given artifact has the same
// identity in every tool and hash-pinned manifests cross-check cleanly.
package identity

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash returns the first 16 hex characters of the SHA-256 digest of content.
//
// The 16-character (64-bit) prefix is a deliberate, frozen contract: it matches
// SkillOpt's skill_hash (skillopt/utils/scoring.py) and the identity used to
// cache evaluations and detect no-op edits. Do not change the algorithm or the
// prefix length without migrating every stored hash.
func Hash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])[:16]
}
