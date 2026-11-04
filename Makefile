.PHONY: check lint test build css

TAILWIND_VERSION := v4.3.3
TAILWIND_OS := $(shell uname -s | sed 's/Darwin/macos/;s/Linux/linux/')
TAILWIND_ARCH := $(shell uname -m | sed 's/x86_64/x64/;s/aarch64/arm64/')
TAILWIND := bin/tailwindcss-$(TAILWIND_VERSION)

check: lint test build

lint:
	golangci-lint run

test: css
	go test ./...

build: css
	CGO_ENABLED=0 go build -o hausbuch ./cmd/hausbuch

css: $(TAILWIND)
	$(TAILWIND) -i internal/web/styles.css -o internal/web/static/app.css --minify

$(TAILWIND):
	mkdir -p bin
	curl -sSfL -o $@ https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_OS)-$(TAILWIND_ARCH)
	chmod +x $@
