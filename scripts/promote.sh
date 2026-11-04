#!/usr/bin/env bash
# Moves :stable onto a release pushed by scripts/release.sh, the newest one when no version is given. Promoting an
# older version rolls :stable back. The tag is moved on ghcr.io itself, for both platforms: nothing is rebuilt.
# Needs Docker with Buildx and the GitHub CLI logged in with the write:packages scope.
set -euo pipefail

cd "$(dirname "$0")/.."

image=ghcr.io/davidepedone/hausbuch

version=${1:-$(git tag -l 'v*' --sort=-v:refname | head -1 | sed 's/^v//')}
if [[ -z "$version" ]]; then
    echo "No release yet. Usage: scripts/promote.sh [YYYY.MM.N]" >&2
    exit 1
fi

gh auth token | docker login ghcr.io -u "$(gh api user --jq .login)" --password-stdin >/dev/null

if ! docker buildx imagetools inspect "$image:$version" >/dev/null 2>&1; then
    echo "$image:$version isn't on ghcr.io." >&2
    exit 1
fi

docker buildx imagetools create -t "$image:stable" "$image:$version"

echo "Done: $image:stable is now $version."
