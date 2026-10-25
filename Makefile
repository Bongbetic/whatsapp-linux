V := $(shell cat VERSION 2>/dev/null || echo 0.0.0-dev)
GO := go
TAGS := sqlite_fts5
LDFLAGS := -s -w -X main.version=$(V)

all: build

build:
	CGO_ENABLED=1 $(GO) build -tags "$(TAGS)" -trimpath -ldflags "$(LDFLAGS)" ./cmd/whatsapp-linux

wlprobe:
	CGO_ENABLED=1 $(GO) build -tags "$(TAGS)" -trimpath ./cmd/wlprobe

test:
	$(GO) test -tags "$(TAGS)" ./...

test-race:
	$(GO) test -tags "$(TAGS)" -race ./internal/store/... ./internal/core/... ./internal/logx/... ./internal/links/...

vet:
	$(GO) vet ./...

fmt:
	gofmt -l .

smoke: build
	xvfb-run -a ./whatsapp-linux --version

clean:
	rm -f whatsapp-linux wlprobe

.PHONY: all build wlprobe test test-race vet fmt smoke clean
