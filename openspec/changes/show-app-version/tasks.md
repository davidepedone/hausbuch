# Tasks

## 1. Build

- [x] 1.1 Declare `version` in `cmd/hausbuch`, set it in the `Dockerfile` from a `VERSION` build argument and pass it from `release.sh`; document in the README where the version shows; verify `docker build --build-arg VERSION=2026.11.99` gives a binary whose pages show that version, and a build without it shows "dev"

## 2. Web interface

- [x] 2.1 Pass the version to `web.NewHandler` and show the footer with the version, "What's new" and "Report a problem" on signed-in pages; verify web tests for "Released build", "Other build", "Not signed in", "Release notes", "Dev build" and "Report", including that the links have no `target`

## 3. Integration

- [ ] 3.1 With `make check` green, at 390px and 1280px and in the app added to a phone's home screen: the footer fits without overflowing, and "What's new" and "Report a problem" open over the app and return to it when closed; verify every step behaves as the spec says

## Workflow follow-up

- Archive the change once merged, which closes issue #34.
