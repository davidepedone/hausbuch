# Built on the build machine's platform and cross-compiled for the target one, so a multi-platform build needs no
# emulation (the CSS doesn't depend on the platform).
FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie@sha256:c65ecfe613b539ac74c62a4c3b2c27e8f8a6076384c8f27e8a3c0f3be17cc385 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH
# The release, which scripts/release.sh passes; any other build shows "dev".
ARG VERSION=dev
RUN make css \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
        go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o /out/hausbuch ./cmd/hausbuch \
    && mkdir /out/data

FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
LABEL org.opencontainers.image.source=https://github.com/davidepedone/hausbuch
COPY --from=build /out/hausbuch /hausbuch
# The image has no shell to create directories; /data comes from the build stage, owned by the nonroot user.
COPY --from=build --chown=nonroot:nonroot /out/data /data
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/hausbuch", "healthcheck"]
ENTRYPOINT ["/hausbuch"]
