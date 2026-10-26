V := $(shell cat VERSION 2>/dev/null || echo 0.0.0-dev)
GO := go
TAGS := sqlite_fts5
LDFLAGS := -s -w -X main.version=$(V)

all: build

build:
	CGO_ENABLED=1 $(GO) build -tags "$(TAGS)" -trimpath -ldflags "$(LDFLAGS)" ./cmd/whatsapp-linux

wlprobe:
	CGO_ENABLED=1 $(GO) build -tags "$(TAGS)" -trimpath ./cmd/wlprobe

wlshot:
	CGO_ENABLED=1 $(GO) build -tags "$(TAGS)" -trimpath ./cmd/wlshot

shots: wlshot
	mkdir -p shots
	xvfb-run -a ./wlshot --page chat --out shots/chat.png
	xvfb-run -a ./wlshot --page list --out shots/list.png
	xvfb-run -a ./wlshot --page login --out shots/login.png
	xvfb-run -a ./wlshot --page chat --dark --out shots/chat-dark.png

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

sounds:
	go run ./tools/gensounds data/sounds

clean:
	rm -f whatsapp-linux wlprobe wlshot

.PHONY: all build wlprobe wlshot shots test test-race vet fmt smoke sounds clean
