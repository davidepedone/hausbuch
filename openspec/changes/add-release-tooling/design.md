# Design

## Context

The image is built by the Dockerfile, which cross-compiles from the build machine's platform, so a multi-platform build
needs no emulation; its `org.opencontainers.image.source` label links the package on ghcr.io to the repository. CI's
`image` job pushes `latest` and the commit SHA for both platforms on every push to `main`. The git history has no tags.

## Goals / Non-Goals

**Goals:**

- One command to release, one to promote, both refusing to run in a state that would publish the wrong thing.
- Versions that sort and read as dates, so it is obvious how old an installed image is.

**Non-Goals:**

- Keeping old CI-built images around: they were never releases.

## Decisions

### Releases cut locally by a script, not by CI

`scripts/release.sh` runs on a developer machine, from `main`. Before anything is built it checks that the branch is
`main`, the working tree is clean, `main` is not behind `origin/main`, `HEAD` is not already released, the GitHub CLI is
logged in with `write:packages`, and `make check` passes. It then:

1. computes the version: `YYYY.MM` of today plus one more than the highest `N` among the git tags `vYYYY.MM.N` of this
   month (so `1` for the first release of a month);
2. builds and pushes in one step with `docker buildx build --platform linux/amd64,linux/arm64 --push`, tagged with the
   version and `candidate`, logging in to ghcr.io with the GitHub CLI's token;
3. creates the annotated tag `vYYYY.MM.N`, pushes it, and creates the GitHub release with generated notes.

The image is pushed before the tag, so a failed build leaves no tag behind and can simply be run again.

Alternative considered: *a workflow triggered by pushing a tag*. No local Docker needed, but the version would be
computed by hand or by a second script anyway, and a failed build would leave a tag pointing at an unpublished
release.

### Promote retags on the registry

`scripts/promote.sh [YYYY.MM.N]` defaults to the newest `v*` tag, checks the version exists on ghcr.io, and runs
`docker buildx imagetools create -t …:stable …:<version>`, which copies the multi-platform manifest list server side:
nothing is pulled or rebuilt, and both platforms move together.

### `candidate` and `stable`

`candidate` always names the newest release; `stable` names the release that has been tried. An installation that
should get releases first follows `candidate`, all others follow `stable`. Two tags are enough for a handful of
installations; a version tag pins an installation to one release.

### Removing `latest` and the SHA tags

Done once, by hand, after the first release is pushed: each package version carrying `latest` or a SHA tag (and no
release tag) is deleted with `gh api -X DELETE /user/packages/container/hausbuch/versions/<id>`. Deleting them only
after a release exists keeps the package, and with it its public visibility and repository link, from disappearing
with its last version.

## Risks / Trade-offs

- [An installation still pulling `latest` stops getting updates, and a fresh pull fails] → The README names
  `stable`; the release notes of the first release say `latest` is gone.
- [A release from a machine whose Docker can't build `linux/arm64`] → The Dockerfile cross-compiles, so Buildx needs no
  emulation; the build fails before anything is tagged if it can't.
- [Two releases cut the same month from different machines could pick the same N] → One maintainer; the tag push
  would fail on the second, after its image was pushed under the same version. Accepted.

## Migration Plan

1. Merge this change: CI stops pushing `latest`.
2. Run `scripts/release.sh`, then `scripts/promote.sh`, so `candidate` and `stable` both exist.
3. Delete the `latest` and SHA-tagged versions from ghcr.io.
4. Installations switch their image to `:stable` (or `:candidate`).
