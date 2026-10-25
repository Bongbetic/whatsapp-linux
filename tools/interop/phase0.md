# Phase 0 interop notes — Gate 0 (tickets WL-0.1..WL-0.3)

Date: 2026-10-26. Machine: Void Linux x86_64, go1.27.3, gtk4
4.22.4, gstreamer 1.28.6, pipewire 1.6.9, pango 1.56.3.

## WL-0.1 — whatsmeow pairing harness

- whatsmeow resolves to `v0.0.0-20261017175727-b2b1e71d45e2`
  (module cache, MPL-2.0 `LICENSE` verified 2026-10-26).
- `Client.GetQRChannel` + `PairPhone` + `Connect` surface
  confirmed in source (`qrchan.go`, `pair-code.go`).
- In-repo harness: `go run ./cmd/wlprobe pair` (throwaway profile
  via `WLPROBE_HOME`). Live QR round-trip requires the owner's
  test SIM (plan §18 prerequisite); recorded accept bar: terminal
  shows QR, `PairSuccess`, echo round-trip < 5 s.

## WL-0.2 — gotk4 pin probe — DONE 2026-10-26

- `v0.3.1` builds, links, and runs (probe binary
  `/tmp/wa-impl/gtkprobe-bin`, "gotk4 links ok") after
  `xbps-install gobject-introspection libgirepository-devel`.
- `v0.4.1` fails: generated code references
  `pango_font_color_get_type` (Pango ≥ 1.57), absent here.
- Pin: `github.com/diamondburned/gotk4/pkg v0.3.1`.
- Remaining accept item (covered by WL-1.0): real window under
  `xvfb-run` for 3 s in CI smoke.

## WL-0.3 — event-coverage run

Harness: `WLPROBE_HOME=$(mktemp -d) go run ./cmd/wlprobe
dump-events`. Tester performs scripted phone actions; each
action → observed event(s) or "none":

| Phone action | Expected event(s) | Observed |
|---|---|---|
| text message | `Message` | live-run pending test SIM |
| image | `Message` + `MediaRetry` if needed | pending |
| voice note | `Message` | pending |
| reaction | `Message` (reaction) | pending |
| poll + vote | `Message` (poll/poll vote) | pending |
| group create | `GroupInfo` / `JoinedGroup` | pending |
| status post | `Message` (status JID) | pending |
| channel follow | `NewsletterJoin` | pending |
| call ring + hangup (unanswered) | `CallOffer`, `CallTerminate` | pending |
| call-offer node dump (own traffic) | saved pcap/node log | pending |

## Gate 0 verdict: GO WITH FORK-LATER

- Messaging/protocol path: **go** (whatsmeow upstream first, D-2).
- Calls: **go-with-fork** — only `RejectCall` exists upstream;
  place/accept + media keys land in the `Bongbetic/whatsmeow`
  fork at WL-5.1. Messaging releases never blocked by calls.
- Fallback D-3 (GTK via C shim) not triggered: gotk4 `v0.3.1`
  probe passed.
