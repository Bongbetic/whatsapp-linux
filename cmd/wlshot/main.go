// Command wlshot: dev-only screenshot harness. Runs the real GUI
// against a fake engine and a seeded throwaway store, renders the
// window to PNG, and exits. Never shipped.
//
//	xvfb-run -a ./wlshot --out shot.png [--page chat|list|login|settings|status|channels|calls] [--dark] [--size 1180x760]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Bongbetic/whatsapp-linux/internal/app"
	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
	"github.com/Bongbetic/whatsapp-linux/internal/settings"
	"github.com/Bongbetic/whatsapp-linux/internal/store"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// Software rendering: deterministic under xvfb, no GPU/Vulkan probing.
	if os.Getenv("GSK_RENDERER") == "" {
		os.Setenv("GSK_RENDERER", "cairo")
	}
	fs := flag.NewFlagSet("wlshot", flag.ContinueOnError)
	out := fs.String("out", "shot.png", "output PNG path")
	page := fs.String("page", "chat", "chat|list|login|settings|status|channels|calls")
	dark := fs.Bool("dark", false, "force dark color scheme")
	size := fs.String("size", "1180x760", "window size WxH")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	w, h, ok := parseSize(*size)
	if !ok {
		fmt.Fprintln(os.Stderr, "bad --size, want WxH")
		return 2
	}
	absOut, err := filepath.Abs(*out)
	if err != nil {
		return 2
	}

	tmp, err := os.MkdirTemp("", "wlshot-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 4
	}
	defer os.RemoveAll(tmp)
	dataDir, cacheDir := filepath.Join(tmp, "data"), filepath.Join(tmp, "cache")
	for _, d := range []string{dataDir, cacheDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 4
		}
	}
	// Keep stray XDG writers (autostart, notify) inside the sandbox.
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "xdgdata"))
	os.Setenv("XDG_CACHE_HOME", cacheDir)

	db, err := store.Open(filepath.Join(dataDir, "app.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "open store:", err)
		return 4
	}
	defer db.Close()
	if err := seed(context.Background(), db); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		return 4
	}

	s := settings.New()
	s.Set(settings.KeyWindowW, w)
	s.Set(settings.KeyWindowH, h)
	s.Set(settings.KeyAutostart, false)
	s.Set(settings.KeyCloseToTray, false)

	eng := &fakeEngine{ev: make(chan bridge.Event, 16), loggedOut: *page == "login"}
	return app.Run(app.Config{
		Engine: eng, DB: db, Settings: s, Log: logx.New(false), Version: "shot",
		DataDir: dataDir, CacheDir: cacheDir,
		Shot: &app.ShotConfig{Out: absOut, Page: *page, Dark: *dark},
	})
}

func parseSize(s string) (int, int, bool) {
	a, b, ok := strings.Cut(s, "x")
	if !ok {
		return 0, 0, false
	}
	w, e1 := strconv.Atoi(a)
	h, e2 := strconv.Atoi(b)
	return w, h, e1 == nil && e2 == nil && w > 0 && h > 0
}
