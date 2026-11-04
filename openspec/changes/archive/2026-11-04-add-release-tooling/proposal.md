# Proposal

## Why

Every push to `main` publishes the image as `latest`, so an installation following that tag gets every merged change,
tried or not, and nothing tells one image from another apart from a commit SHA. Installations need named releases, and
a way to try a release on one installation before the others follow it.

## What Changes

- CI stops publishing the image; it keeps running `make check` on every push.
- A release script cuts a release from an up-to-date, clean `main` that passes `make check`: it picks the next CalVer
  version `YYYY.MM.N` (N restarts every month), builds the image for `linux/amd64` and `linux/arm64`, pushes it to
  `ghcr.io/davidepedone/hausbuch` as that version and as `candidate`, tags the commit `vYYYY.MM.N` and creates the
  GitHub release.
- A promote script moves `stable` onto a released version, the newest one when none is given; promoting an older
  version rolls `stable` back. The tag is moved on the registry, nothing is rebuilt.
- **BREAKING**: the `latest` tag and the commit-SHA tags are deleted from the registry. Installations follow `stable`
  (or `candidate`, to try releases first); the README says so and its example uses `stable`.

### Out of scope

- Release notes beyond GitHub's generated ones, and showing the version inside the application.
- Releasing from CI (tag-triggered workflows): releases are cut from a developer machine.
- Automatic updates of installations (e.g. watchtower setup): the README only names the tags to follow.
- Signing images or producing SBOMs.

## Capabilities

### New Capabilities

None: this change is about how the image is built and published, not about what the system does.

### Modified Capabilities

None.

## Impact

- `.github/workflows/build.yml` loses its `image` job.
- New `scripts/release.sh` and `scripts/promote.sh`; they need Docker with Buildx and the GitHub CLI logged in with
  the `write:packages` scope.
- The registry keeps only release versions, `candidate` and `stable`.
- README: which tag to run, and how releases are made.
