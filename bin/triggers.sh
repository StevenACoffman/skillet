#!/usr/bin/env bash
# triggers.sh — report which of skillet's deferred decisions are now decidable.
#
# Thin wrapper around `invigilator check` (github.com/StevenACoffman/invigilator),
# which reads holds.toml and counts how many of the given repositories satisfy each
# hold's condition.
#
# The roots are arguments and there are no defaults. Which checkouts constitute
# "the family" is a fact about one machine, and a committed default would be one
# person's layout imposed on everyone else's.
#
# Usage:
#   bin/triggers.sh ROOT...
#   INVIGILATOR_QUIET=1 bin/triggers.sh ROOT...    # only holds that are met
#   INVIGILATOR_JSON=1  bin/triggers.sh ROOT...    # machine-readable
#
# Exit: 0 nothing met, 2 at least one hold met, 1 an error.
#
# A met condition is not a failure. It means a decision this repository deferred is
# now answerable, which is the whole point — every criterion in TODO.md was
# previously a note to a reader who had to already be looking for it.
set -euo pipefail

if [ "$#" -eq 0 ]; then
    cat >&2 <<'USAGE'
usage: bin/triggers.sh ROOT...

  ROOT   a consumer checkout to search

Example, from a machine where the family sits side by side:

  bin/triggers.sh ../exegesis ../skillsaw ../agentic-dev-harness ../canonizer ../gnosis
USAGE
    exit 1
fi

# has_cmd NAME — true if NAME is an executable file on $PATH.
# Ignores shell functions, aliases, and builtins of the same name.
has_cmd() {
    if [ -n "${ZSH_VERSION:-}" ]; then
        builtin whence -p -- "$1" >/dev/null 2>&1
    elif [ -n "${BASH_VERSION:-}" ]; then
        builtin type -P -- "$1" >/dev/null 2>&1
    else
        command -v -- "$1" >/dev/null 2>&1
    fi
}

here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"

# Resolve every root to an absolute path before changing directory, so a relative
# argument means what the caller meant rather than what the last cd made it mean.
roots=()
for r in "$@"; do
    if [ ! -d "$r" ]; then
        echo "triggers.sh: not a directory: $r" >&2
        exit 1
    fi
    roots+=("$(cd -- "$r" && pwd)")
done

cd -- "$here"

if has_cmd invigilator; then
    exec invigilator check -f "$here/holds.toml" "${roots[@]}"
fi

# Not installed. Run from a checkout if one was named, so this works before the first
# release. go run needs the module's own directory as the working directory, which is
# why the paths above were made absolute.
if [ -n "${INVIGILATOR_SRC:-}" ] && has_cmd go; then
    cd -- "$INVIGILATOR_SRC"
    exec go run . check -f "$here/holds.toml" "${roots[@]}"
fi

cat >&2 <<'MISSING'
triggers.sh: invigilator is not on PATH.

  go install github.com/StevenACoffman/invigilator@latest

or point INVIGILATOR_SRC at a local checkout to run it from source:

  INVIGILATOR_SRC=../invigilator bin/triggers.sh ROOT...
MISSING
exit 1
