// Command whatsapp-linux: GUI entry; flags per plan §11.
//
//	whatsapp-linux [--hidden] [--reset-profile [--full]] [--version]
//	               [--debug] [whatsapp://…|https://wa.me/…]
//
// Exit codes: 0 ok, 2 bad args, 3 already-running / store locked,
// 4 store corrupt.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/links"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
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

	var intent *links.Intent
	for _, a := range fs.Args() {
		in, err := links.Parse(a)
		if err != nil {
			log.Error().Str("arg", logx.Scrub(a)).Msg("bad URI argument")
			return 2
		}
		intent = &in
	}
	_ = intent // handed to the app shell at WL-2.1; parsed+validated here

	if *showVer {
		fmt.Println("whatsapp-linux", version)
		return 0
	}
	if *reset {
		return resetProfile(*full)
	}

	dataDir := profileDataDir()
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		log.Error().Err(err).Msg("profile dir")
		return 4
	}

	app := gtk.NewApplication("com.bongbetic.WhatsAppLinux", gio.ApplicationHandlesOpen)
	app.ConnectActivate(func() { activate(app, *hidden) })
	// TODO WL-2.1: single-instance URI handoff via "open" signal.
	if code := app.Run(nil); code != 0 {
		log.Error().Int("code", code).Msg("application exited non-zero")
		return 3
	}
	return 0
}

// activate builds the Phase-1 placeholder window (WL-1.0 accept:
// `make` builds an empty GUI that opens a window). WL-2.1 replaces
// this body with the internal/app shell.
func activate(app *gtk.Application, hidden bool) {
	win := gtk.NewApplicationWindow(app)
	win.SetTitle("WhatsApp (unofficial)")
	win.SetDefaultSize(1180, 760)
	label := gtk.NewLabel("whatsapp-linux — pairing and chat UI lands in Phase 2 (WL-2.x).")
	win.SetChild(label)
	if !hidden {
		win.Show()
	}
}

// resetProfile implements --reset-profile (§11).
func resetProfile(full bool) int {
	dir := profileDataDir()
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
