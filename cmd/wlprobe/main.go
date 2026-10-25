// Command wlprobe: headless protocol harness, dev only (plan §11).
// Never shipped. Profile dir from WLPROBE_HOME (throwaway).
//
//	wlprobe pair            QR login (ASCII QR on stdout)
//	wlprobe send <jid> <text>  send one text message
//	wlprobe dump-events     log every event until interrupted
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/skip2/go-qrcode"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/core"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: wlprobe pair|send <jid> <text>|dump-events|call-probe|export <app.db> <jid> <txt|json>")
		return 2
	}
	home := os.Getenv("WLPROBE_HOME")
	if home == "" {
		fmt.Fprintln(os.Stderr, "WLPROBE_HOME must point at a throwaway profile dir")
		return 2
	}
	log := logx.New(true)
	eng, err := core.New(log, home, nil)
	if err != nil {
		log.Error().Err(err).Msg("new engine")
		return 4
	}
	ctx := context.Background()
	if err := eng.Open(ctx, home); err != nil {
		log.Error().Err(err).Msg("open")
		return 4
	}
	defer eng.Stop()

	switch args[0] {
	case "export":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: wlprobe export <app.db> <jid> <txt|json>")
			return 2
		}
		return doExport(args[1], args[2], args[3], log)
	case "pair":
		return doPair(ctx, eng)
	case "send":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: wlprobe send <jid> <text>")
			return 2
		}
		if err := eng.Start(ctx); err != nil {
			log.Error().Err(err).Msg("connect")
			return 3
		}
		id, err := eng.SendText(ctx, parseJID(args[1]), strings.Join(args[2:], " "), bridge.SendOpts{})
		if err != nil {
			log.Error().Err(err).Msg("send")
			return 3
		}
		fmt.Println("sent", id)
		return 0
	case "dump-events":
		return doDump(ctx, eng, log)
	case "call-probe":
		return doCallProbe(ctx, eng, log, home)
	default:
		fmt.Fprintln(os.Stderr, "unknown subcommand", args[0])
		return 2
	}
}

func doPair(ctx context.Context, eng bridge.Engine) int {
	if err := eng.Start(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		return 3
	}
	codes, err := eng.PairQR(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pair:", err)
		return 3
	}
	for {
		select {
		case code, ok := <-codes:
			if !ok {
				return 3
			}
			if err := printQR(code.Code); err != nil {
				fmt.Fprintln(os.Stderr, "qr render:", err)
				return 3
			}
		case ev := <-eng.Events():
			if done, ok := ev.(bridge.PairDone); ok {
				fmt.Println("pair success:", done.OK)
				if done.OK {
					return 0
				}
				return 3
			}
		}
	}
}

// printQR renders one pairing code as terminal blocks (WL-0.1).
func printQR(code string) error {
	qr, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		return err
	}
	for _, row := range qr.Bitmap() {
		var sb strings.Builder
		for _, black := range row {
			if black {
				sb.WriteString("██")
			} else {
				sb.WriteString("  ")
			}
		}
		fmt.Println(sb.String())
	}
	return nil
}

// doCallProbe captures the WL-5.0 signaling evidence: every call
// event (offer/accept/transport/terminate with IDs and peers) is
// appended to $WLPROBE_HOME/calls.log for the media plan A/B
// decision. Run while placing/ringing calls from the phone.
func doCallProbe(ctx context.Context, eng bridge.Engine, log zerolog.Logger, home string) int {
	if err := eng.Start(ctx); err != nil {
		log.Error().Err(err).Msg("connect")
		return 3
	}
	f, err := os.OpenFile(home+"/calls.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		log.Error().Err(err).Msg("calls.log")
		return 4
	}
	defer f.Close()
	fmt.Fprintln(f, "--- call-probe started", time.Now().UTC().Format(time.RFC3339))
	for ev := range eng.Events() {
		ce, ok := ev.(bridge.CallEvent)
		if !ok {
			continue
		}
		line := fmt.Sprintf("%s id=%s dir=%s media=%s stage=%s chat=%s",
			time.Now().UTC().Format(time.RFC3339), ce.ID, ce.Direction, ce.Media, ce.Stage, ce.Chat)
		fmt.Println(line)
		fmt.Fprintln(f, line)
	}
	return 0
}

// doExport dumps one chat's history (WL-6.2 proof path).
func doExport(dbPath, jid, format string, log zerolog.Logger) int {
	db, err := appstore.Open(dbPath)
	if err != nil {
		log.Error().Err(err).Msg("open app.db")
		return 4
	}
	defer db.Close()
	if err := db.ExportChat(context.Background(), jid, format, os.Stdout); err != nil {
		log.Error().Err(err).Msg("export")
		return 3
	}
	return 0
}

func doDump(ctx context.Context, eng bridge.Engine, log zerolog.Logger) int {
	if err := eng.Start(ctx); err != nil {
		log.Error().Err(err).Msg("connect")
		return 3
	}
	for ev := range eng.Events() {
		log.Info().Str("kind", ev.Kind()).Time("at", time.Now()).Msg("event")
	}
	return 0
}

func parseJID(s string) bridge.JID {
	if i := strings.IndexByte(s, '@'); i >= 0 {
		return bridge.JID{User: s[:i], Server: s[i+1:]}
	}
	return bridge.JID{User: s, Server: "s.whatsapp.net"}
}
