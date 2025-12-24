# Makefile to pull latest tag in current repo and build Go binary

REPO_URL := git@github.com/72nd/schipla.git  # replace OWNER/REPO
BRANCH   ?= v2                             # git dev branch
GO_OUTPUT := shifts
GO_MAIN := cmd/shifts/main.go

# Mode: tag | dev
MODE ?= dev

# Repository configuration
REPO_URL := https://github.com/OWNER/REPO.git   # replace OWNER/REPO
BRANCH   ?= main                                # dev branch

# Mode: tag | dev
MODE ?= tag

# Go build configuration
GO_OUTPUT := shifts
GO_MAIN   := cmd/shifts/main.go

# Go tooling
GO_BIN := $(shell go env GOPATH)/bin
SWAG   := $(GO_BIN)/swag

.PHONY: all update docs build clean status tools

all: update docs build

# Clone if needed, then switch based on MODE
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

# Ensure swag is installed
tools:
	@if [ ! -x "$(SWAG)" ]; then \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	else \
		echo "swag already installed"; \
	fi

# Build Swagger API docs
docs: tools
	@echo "Generating Swagger docs..."
	@$(SWAG) init \
		-g $(GO_MAIN) \
		-o docs \
		--parseDependency \
		--parseInternal

# Build Go binary with CGO enabled
build:
	@echo "Building $(GO_OUTPUT) (MODE=$(MODE), CGO_ENABLED=1)..."
	@CGO_ENABLED=1 go build -o $(GO_OUTPUT) $(GO_MAIN)
	@echo "Build complete: $(GO_OUTPUT)"

# Show current git state
status:
	@echo "MODE=$(MODE)"
	@echo "BRANCH=$(BRANCH)"
	@git --no-pager describe --tags --dirty --always
	@git --no-pager branch --show-current

# Clean generated files
clean:
	@echo "Cleaning up..."
	@rm -f $(GO_OUTPUT)
	@rm -rf docs

