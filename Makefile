# Git Flow Plus build automation.
#
# Common targets:
#   make build     build a local binary for the host platform into bin/
#   make test      run the full test suite
#   make lint      run golangci-lint
#   make fmt       check gofmt formatting (fails if anything is unformatted)
#   make vet       run go vet
#   make check     fmt + vet + lint + test (what CI runs)
#   make dist      cross-compile release binaries into dist/ (6 platforms)
#   make package   dist, then package dist/ into dist/archives/ (zip/tar.gz)
#   make clean     remove bin/ and dist/
#
# On Windows without a `make` on PATH, use scripts\build.ps1 and
# scripts\package.ps1 directly instead — see DeveloperGuide.md.

CLI_PKG := github.com/hulhub/git-flow-plus/internal/cli
CMD_PKG := ./cmd/git-flow-plus

VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)
BUILD_NUMBER ?= dev
GIT_COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GIT_BRANCH   ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
BUILD_DATE   ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(CLI_PKG).Version=$(VERSION) \
	-X $(CLI_PKG).BuildNumber=$(BUILD_NUMBER) \
	-X $(CLI_PKG).GitCommit=$(GIT_COMMIT) \
	-X $(CLI_PKG).GitBranch=$(GIT_BRANCH) \
	-X $(CLI_PKG).BuildDate=$(BUILD_DATE)

.PHONY: build test lint fmt vet check dist package clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/git-flow $(CMD_PKG)

test:
	go test ./... -cover

lint:
	golangci-lint run ./...

fmt:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needs to be run on:"; echo "$$unformatted"; exit 1; \
	fi

vet:
	go vet ./...

check: fmt vet lint test

dist:
	./scripts/build.sh

package: dist
	./scripts/package.sh

clean:
	rm -rf bin dist
