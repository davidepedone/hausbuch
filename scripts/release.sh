#!/usr/bin/env bash
# Cuts the next release from main: CalVer version YYYY.MM.N (N restarts every month), image for linux/amd64 and
# linux/arm64 pushed to ghcr.io as that version and as :candidate, git tag vYYYY.MM.N and GitHub release.
# scripts/promote.sh then moves :stable onto it once it has been tried.
# Needs Docker with Buildx and the GitHub CLI logged in with the write:packages scope.
set -euo pipefail

cd "$(dirname "$0")/.."

image=ghcr.io/davidepedone/hausbuch

fail() {
    echo "$1" >&2
    exit 1
}

# next_version prints the version following the git tags of this month: YYYY.MM.1 when there are none.
next_version() {
    local month last
    month=$(date +%Y.%m)
    last=$(git tag -l "v$month.*" | sed "s/^v$month\.//" | sort -n | tail -1)
    echo "$month.$((${last:-0} + 1))"
}

[[ "$(git branch --show-current)" == main ]] || fail "Releases are cut from main."
[[ -z "$(git status --porcelain)" ]] || fail "Uncommitted changes: commit or stash them first."

git fetch -q origin main
git merge-base --is-ancestor origin/main HEAD || fail "main is behind origin/main: pull first."

released=$(git tag --points-at HEAD 'v*')
[[ -z "$released" ]] || fail "HEAD is already released as $released."

if ! command -v gh >/dev/null || ! gh auth status >/dev/null 2>&1; then
    fail "The GitHub CLI is needed: brew install gh && gh auth login"
fi
gh auth status 2>&1 | grep -q "write:packages" ||
    fail "Pushing to ghcr.io needs the write:packages scope: gh auth refresh -s write:packages"

make check || fail "make check fails: nothing released."

version=$(next_version)
echo "Releasing $version ($(git rev-parse --short HEAD))"

gh auth token | docker login ghcr.io -u "$(gh api user --jq .login)" --password-stdin >/dev/null
# The image is pushed before the tag is created: a failed build leaves nothing behind and can be run again.
docker buildx build --platform linux/amd64,linux/arm64 --push -t "$image:$version" -t "$image:candidate" .

git tag -a "v$version" -m "Release $version"
git push -q origin "v$version"
gh release create "v$version" --verify-tag --title "$version" --generate-notes

echo "Done: $image:$version is out as :candidate. Once tried, scripts/promote.sh makes it :stable."
