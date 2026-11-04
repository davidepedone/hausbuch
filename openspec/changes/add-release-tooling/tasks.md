# Tasks

## 1. Scripts and CI

- [ ] 1.1 Remove the `image` job from `.github/workflows/build.yml`; verify the branch's CI run only has the `check` job and passes
- [x] 1.2 Add `scripts/release.sh` with the checks, CalVer version, multi-platform build and push, git tag and GitHub release from the design; verify each refusal (not on `main`, dirty tree, behind `origin/main`, `HEAD` already released, missing `write:packages`, failing `make check`) exits non-zero before building, and that the version is computed correctly with no tags, with tags of an earlier month and with tags of this month
- [x] 1.3 Add `scripts/promote.sh` defaulting to the newest release tag and refusing a version that isn't on ghcr.io; verify the refusal, and run `shellcheck` on both scripts
- [x] 1.4 Update the README: the example compose uses `:stable`, a section on the `candidate` and `stable` tags and version tags, and a development section on releasing and promoting; verify by reading it against the scripts

## 2. First release

- [ ] 2.1 After merge, run `scripts/release.sh` on `main`; verify with `docker buildx imagetools inspect` that `:<version>` and `:candidate` exist for `linux/amd64` and `linux/arm64`, and that the git tag and GitHub release exist
- [ ] 2.2 Run `scripts/promote.sh`; verify `:stable` has the same digest as `:<version>`
- [ ] 2.3 Delete the package versions tagged `latest` or with a commit SHA; verify the package lists only the release version with `candidate` and `stable`, is still public, and `docker pull ghcr.io/davidepedone/hausbuch:stable` works logged out
