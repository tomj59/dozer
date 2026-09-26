# dozer build helpers. `make` builds bin/dozer for this machine.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
TARGETS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: build test bench dist clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/dozer ./cmd/dozer

test:
	go vet ./...
	go test ./...

bench:
	go test ./internal/emu -run x -bench . -benchmem

dist:
	@mkdir -p dist
	@for t in $(TARGETS); do \
		os=$${t%/*}; arch=$${t#*/}; \
		echo "building dist/dozer-$$os-$$arch"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" \
			-o dist/dozer-$$os-$$arch ./cmd/dozer || exit 1; \
	done

clean:
	rm -rf bin dist
