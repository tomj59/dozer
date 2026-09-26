# dozer build helpers. `make` builds bin/dozer for this machine.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
TARGETS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: build test bench dist install cases clean
PREFIX ?= /usr/local

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

# Put dozer on your PATH (default /usr/local/bin; e.g. make install PREFIX=~/.local).
install: build
	install -d $(PREFIX)/bin
	install -m 755 bin/dozer $(PREFIX)/bin/dozer

# Validate every packaged test case in tests/cases (they are the source:
# edit one with ./X.sh --show-config > x.yaml, then dozer -c x.yaml --package X.sh).
cases: build
	@for f in tests/cases/*.sh; do \
		printf '%-40s ' "$$f"; bin/dozer --check -c $$f </dev/null | sed -n 2p || exit 1; \
	done

clean:
	rm -rf bin dist
