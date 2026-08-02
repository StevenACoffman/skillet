#!/usr/bin/env bash
# Tag HEAD of the default branch as a signed release and push the tag. Pushing
# the tag triggers .github/workflows/release.yml, which runs goreleaser to build
# the source tarball, its SBOM, a checksum, and a keyless cosign signature.
#
# usage: ./bin/release.sh vX.Y.Z
set -euo pipefail

OWNER=StevenACoffman
REPO=skillet
BRANCH=main

[ $# -eq 1 ] || { echo "usage: ./bin/release.sh vX.Y.Z" >&2; exit 1; }
VERSION=$1

# Version must be a vX.Y.Z semver tag (Go modules require the leading v).
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "error: version must look like vX.Y.Z (got '$VERSION')" >&2
  exit 1
fi

# git tag -s needs a signing key; fail early with a clear message instead of a
# raw gpg error mid-release.
if [ -z "$(git config --get user.signingkey || true)" ]; then
  echo "error: no git signing key configured (user.signingkey); 'git tag -s' needs one" >&2
  exit 1
fi

# Refuse to release with a dirty tree — the tag must capture committed state.
if ! git diff-index --quiet HEAD --; then
  echo "error: uncommitted changes on HEAD, aborting" >&2
  exit 1
fi

# Refuse to reuse a tag, locally or on the remote.
git fetch --tags origin
if git rev-parse -q --verify "refs/tags/${VERSION}" >/dev/null; then
  echo "error: tag ${VERSION} already exists locally" >&2
  exit 1
fi
if git ls-remote --exit-code --tags origin "refs/tags/${VERSION}" >/dev/null 2>&1; then
  echo "error: tag ${VERSION} already exists on origin" >&2
  exit 1
fi

# Release from the tip of an up-to-date default branch (no detached HEAD).
git checkout "$BRANCH"
git pull --ff-only origin "$BRANCH"

# Show the current latest release for context (handles the first-release case).
latest=$(gh release view --repo "${OWNER}/${REPO}" --json tagName -q .tagName 2>/dev/null || echo "none")
echo "Current latest release: ${latest}"
read -r -p "Tag $(git rev-parse --short HEAD) on ${BRANCH} as ${VERSION} and push it? [y/N] " reply
[[ "$reply" == [yY] ]] || { echo "aborted"; exit 1; }

git tag -s "${VERSION}" -m "${VERSION}"
git push origin "${VERSION}"

echo "Pushed ${VERSION}; the release workflow will build the signed SBOM."
echo "Now write the release notes: https://github.com/${OWNER}/${REPO}/releases"
