VERSION ?= dev

.PHONY: build install test check fmt vet docs docs-check

build:
	go build -trimpath -ldflags "-X main.version=$(VERSION)" -o bin/elevenlabs ./cmd/elevenlabs

install:
	go install -trimpath -ldflags "-X main.version=$(VERSION)" ./cmd/elevenlabs

test:
	@set -eu; engine_dir=$$(mktemp -d); trap 'rm -rf "$$engine_dir"' EXIT; \
	module_dir=$$(go list -m -f '{{.Dir}}' github.com/Microck/galleton); \
	(cd "$$module_dir" && go build -trimpath -o "$$engine_dir/galleton" ./cmd/galleton); \
	ELEVENLABS_TEST_ENGINE="$$engine_dir/galleton" go test ./... -count=1

vet:
	go vet ./...

check: vet test docs-check
	test -z "$$(gofmt -l cmd internal scripts/embed.go)"

fmt:
	gofmt -w cmd internal scripts/embed.go

docs:
	go run ./cmd/docsgen

docs-check:
	go run ./cmd/docsgen --check
