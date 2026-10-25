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
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: wlprobe pair|send <jid> <text>|dump-events|call-probe")
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
		fmt.Println("call-probe lands at WL-5.0 (packet capture of own calls)")
		return 0
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
