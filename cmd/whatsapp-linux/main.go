// Command whatsapp-linux: GUI entry; flags per plan §11.
//
//	whatsapp-linux [--hidden] [--reset-profile [--full]] [--version]
//	               [--debug] [whatsapp://…|https://wa.me/…]
//
// Exit codes: 0 ok, 2 bad args, 3 already-running / engine failure,
// 4 store corrupt.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Bongbetic/whatsapp-linux/internal/app"
	"github.com/Bongbetic/whatsapp-linux/internal/core"
	"github.com/Bongbetic/whatsapp-linux/internal/links"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
	"github.com/Bongbetic/whatsapp-linux/internal/settings"
	"github.com/Bongbetic/whatsapp-linux/internal/store"
)

var version = "0.0.0-dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("whatsapp-linux", flag.ContinueOnError)
	hidden := fs.Bool("hidden", false, "start hidden in tray")
	reset := fs.Bool("reset-profile", false, "delete app.db+media (keep session.db)")
	full := fs.Bool("full", false, "with --reset-profile: also log out and delete session.db")
	showVer := fs.Bool("version", false, "print version")
	debug := fs.Bool("debug", false, "debug logging")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	log := logx.New(*debug)

	var rawURIs []string
	var intents []links.Intent
	for _, a := range fs.Args() {
		in, err := links.Parse(a)
		if err != nil {
			log.Error().Str("arg", logx.Scrub(a)).Msg("bad URI argument")
			return 2
		}
		rawURIs = append(rawURIs, a)
		intents = append(intents, in)
	}
	// WL-2.1 single instance: forward URIs to the owner, else exit.
	primary, code, linkCh, stop := app.ClaimInstance(rawURIs)
	if !primary {
		return code
	}
	defer stop()

	if *showVer {
		fmt.Println("whatsapp-linux", version)
		return 0
	}
	dataDir := profileDataDir()
	if *reset {
		return resetProfile(dataDir, *full)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		log.Error().Err(err).Msg("profile dir")
		return 4
	}

	db, err := store.Open(filepath.Join(dataDir, "app.db"))
	if err != nil {
		log.Error().Err(err).Msg("open app.db")
		return 4
	}
	defer db.Close()

	eng, err := core.New(log, dataDir, db)
	if err != nil {
		log.Error().Err(err).Msg("new engine")
		return 4
	}
	if err := eng.Open(context.Background(), dataDir); err != nil {
		log.Error().Err(err).Msg("open session")
		return 4
	}
	defer eng.Stop()

	s := settings.New() // GSettings backend attaches when schema installed (WL-2.6)
	return app.Run(app.Config{Engine: eng, DB: db, Settings: s, Log: log, Version: version, Hidden: *hidden, DataDir: dataDir, CacheDir: cacheDir(), Links: intents, LinkCh: linkCh})
}

func cacheDir() string {
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "whatsapp-linux")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "whatsapp-linux")
}

// resetProfile implements --reset-profile (§11).
func resetProfile(dir string, full bool) int {
	remove := []string{"app.db", "app.db-wal", "app.db-shm", "media", "avatars", "stickers"}
	if full {
		remove = append(remove, "session.db", "session.db-wal", "session.db-shm")
	}
	for _, n := range remove {
		os.RemoveAll(filepath.Join(dir, n))
	}
	return 0
}

func profileDataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "whatsapp-linux")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "whatsapp-linux")
}
