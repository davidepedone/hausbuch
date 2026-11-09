# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. The current state that shapes the
approach:

- `scripts/release.sh` computes the version (`YYYY.MM.N`), builds the image with `docker buildx build`, then tags the
  commit `vYYYY.MM.N` and creates the GitHub release with generated notes. The image is built before the tag exists.
- The `Dockerfile` builds the binary with `go build -trimpath -ldflags="-s -w"`; `make build` builds it without flags.
- `web.NewHandler` takes the stores and the default language; the layout renders every page through `view`, which
  knows whether a user is signed in.
- The repository's address already appears in the image, as `org.opencontainers.image.source`.

## Goals / Non-Goals

**Goals:**

- The version comes from the release script alone: nothing to edit by hand when releasing.

**Non-Goals:**

- Reading the version from git or from Go's build information: the image is built from a copy of the sources, without
  `.git`, and before the tag is created.

## Decisions

### A linker variable set from a build argument

`cmd/hausbuch` declares `var version = "dev"`. The `Dockerfile` takes `ARG VERSION=dev` and adds
`-X main.version=$VERSION` to the linker flags; `release.sh` passes `--build-arg VERSION=$version`. Any other build
(`make build`, `go run`, a plain `docker build`) keeps "dev".

*Alternative considered*: an environment variable set in the image. It could be overridden at run time and would show
a version the binary isn't.

### The footer in the layout

`web.NewHandler` takes the version, and `view` carries it to the layout, which shows the footer under `<main>` only
when a user is signed in. The links are built from one repository address in `internal/web`:
`/releases/tag/v<version>`, `/releases` for "dev", and `/issues/new/choose`. They are plain links, without
`target="_blank"`: in a home-screen app, iOS and Android open another site's page in a browser view over the app.

## Risks / Trade-offs

- [A fork's build links to this repository's releases and issues] → Accepted: the image is published from this
  repository; a fork changes the one address in `internal/web`.
- [The behaviour of links in a home-screen app depends on the phone's system] → Checked on a phone before merging
  (task 3.1).
