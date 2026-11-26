# Makefile for building the Shifts application

REPO_URL  := git@github.com/72nd/schipla.git
BRANCH    ?= v2
GO_OUTPUT := shifts
GO_MAIN   := cmd/shifts/main.go

# Mode: tag | dev
MODE ?= tag

# Go tooling
GO_BIN := $(shell go env GOPATH)/bin
SWAG   := $(GO_BIN)/swag

.PHONY: all install update docs check-version build dist clean status tools help

## all: Update to the latest release and rebuild (shortcut for: make update && make install)
all: update install

## install: Generate Swagger docs and compile the binary
install: docs build

## update: Fetch and switch to the latest release tag (use MODE=dev for the dev branch)
update:
	@if [ ! -d ".git" ]; then \
		echo "No git repo found. Cloning..."; \
		git clone $(REPO_URL) .; \
	fi
	@echo "Fetching latest changes and tags..."
	@git fetch --all --tags
	@if [ "$(MODE)" = "dev" ]; then \
		echo "Switching to dev mode (branch: $(BRANCH))..."; \
		git checkout $(BRANCH); \
		git pull --ff-only origin $(BRANCH); \
	else \
		LATEST_TAG=$$(git describe --tags `git rev-list --tags --max-count=1`); \
		echo "Switching to latest tag ($$LATEST_TAG)..."; \
		git checkout $$LATEST_TAG; \
	fi

## tools: Install the Swag CLI in the version from go.mod, built with the current Go toolchain
# swag parses the standard library with the go/parser it was built with, so a
# binary built by an older Go fails on newer stdlib syntax. Reinstall whenever
# the swag version or the Go version of the binary differs.
tools:
	@WANT_SWAG=$$(go list -m -f '{{.Version}}' github.com/swaggo/swag); \
	WANT_GO=$$(go env GOVERSION); \
	HAVE=$$( [ -x "$(SWAG)" ] && go version -m "$(SWAG)" 2>/dev/null ); \
	HAVE_GO=$$(echo "$$HAVE" | head -1 | awk '{print $$2}'); \
	HAVE_SWAG=$$(echo "$$HAVE" | awk '$$1=="mod" && $$2=="github.com/swaggo/swag" {print $$3}'); \
	if [ "$$HAVE_SWAG" != "$$WANT_SWAG" ] || [ "$$HAVE_GO" != "$$WANT_GO" ]; then \
		echo "Installing swag $$WANT_SWAG with $$WANT_GO..."; \
		go install github.com/swaggo/swag/cmd/swag@$$WANT_SWAG; \
	else \
		echo "swag $$HAVE_SWAG ($$HAVE_GO) already installed"; \
	fi

## docs: Generate Swagger API documentation (runs tools first)
docs: tools
	@echo "Generating Swagger docs..."
	@$(SWAG) init \
		-g $(GO_MAIN) \
		-o docs \
		--parseDependency \
		--parseInternal

## check-version: Fail if internal/web/version.go differs from the newest CHANGELOG.md release heading
check-version:
	@CL=$$(grep -oE '^## v[0-9]+\.[0-9]+\.[0-9]+' CHANGELOG.md | tail -1 | cut -c4-); \
	V=$$(grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' internal/web/version.go); \
	if [ "$$CL" != "$$V" ]; then echo "Version mismatch: version.go=$$V, CHANGELOG.md=$$CL"; exit 1; fi

## build: Compile the binary (CGO_ENABLED=1 required for SQLite)
build: check-version
	@echo "Building $(GO_OUTPUT)..."
	@CGO_ENABLED=1 go build -o $(GO_OUTPUT) $(GO_MAIN)
	@echo "Build complete: ./$(GO_OUTPUT)"

## dist: Cross-compile for linux/amd64, linux/arm64, windows/amd64, windows/arm64 into dist/
## dist:   Requires cross-compilers: aarch64-linux-gnu-gcc, x86_64-w64-mingw32-gcc, aarch64-w64-mingw32-gcc
dist:
	@mkdir -p dist
	@echo "Building linux/amd64..."
	@CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build -o dist/$(GO_OUTPUT)-linux-amd64 $(GO_MAIN)
	@echo "Building linux/arm64..."
	@CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC=aarch64-linux-gnu-gcc \
		go build -o dist/$(GO_OUTPUT)-linux-arm64 $(GO_MAIN)
#	@echo "Building windows/amd64..."
#	@CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
#		CC=x86_64-w64-mingw32-gcc \
#		go build -o dist/$(GO_OUTPUT)-windows-amd64.exe $(GO_MAIN)
#	@echo "Building windows/arm64..."
#	@CGO_ENABLED=1 GOOS=windows GOARCH=arm64 \
#		CC=aarch64-w64-mingw32-gcc \
#		go build -o dist/$(GO_OUTPUT)-windows-arm64.exe $(GO_MAIN)
	@echo "dist/ contents:"; ls -lh dist/

## status: Show current git state and active branch
status:
	@echo "MODE=$(MODE)"
	@echo "BRANCH=$(BRANCH)"
	@git --no-pager describe --tags --dirty --always
	@git --no-pager branch --show-current

## clean: Remove the compiled binary, generated docs/ folder, and dist/
clean:
	@echo "Cleaning up..."
	@rm -f $(GO_OUTPUT)
	@rm -rf docs dist

## help: List all available targets
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/## /  make /'
