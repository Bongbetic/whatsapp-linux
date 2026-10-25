V := $(shell cat VERSION 2>/dev/null || echo 0.0.0-dev)
GO := go
LDFLAGS := -s -w -X main.version=$(V)

all: build

build:
	CGO_ENABLED=1 $(GO) build -trimpath -ldflags "$(LDFLAGS)" ./cmd/whatsapp-linux

wlprobe:
	CGO_ENABLED=1 $(GO) build -trimpath ./cmd/wlprobe

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./internal/store/... ./internal/core/... ./internal/logx/... ./internal/links/...

vet:
	$(GO) vet ./...

fmt:
	gofmt -l .

smoke: build
	xvfb-run -a ./whatsapp-linux --version

clean:
	rm -f whatsapp-linux wlprobe

.PHONY: all build wlprobe test test-race vet fmt smoke clean
