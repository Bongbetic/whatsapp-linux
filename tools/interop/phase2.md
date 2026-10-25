# Phase 2 interop notes — Gate 2 checklist (WL-2.1..WL-2.8)

Two-account script (A = Linux build, B = phone). `tools/interop/phase1.md`
must be green first.

## WL-2.1 shell + login

- [ ] Fresh profile: window 1180×760, QR shown; B scans; chat list appears.
- [ ] Narrow window (<700 px): panes usable (collapse polish post-0.1).
- [ ] `--hidden` starts without a visible window; tray keeps process alive.
- [ ] Ctrl+N opens new-chat; Ctrl+, opens settings.

## WL-2.2 chat list + convo read path

- [ ] 200-chat account: list newest-first, unread pills, mute/pin marks.
- [ ] Search filters live; archived toggle hides/shows archived.
- [ ] Open chat: last 50 messages, date order, text + image thumbs + system lines.
- [ ] In-chat search (Ctrl+Shift+F): FTS results section.

## WL-2.3 composer send path

- [ ] Enter sends / Shift+Enter newline (and flipped with enter-to-send off).
- [ ] Reply banner quotes; attachment button present (picker at WL-3.x).
- [ ] Draft survives chat switch (500 ms persist).

## WL-2.4 message actions

- [ ] React, edit own, revoke, forward picker, star, multi-select,
      context menus; poll create/vote/results render.

## WL-2.5 notifications + tray

- [ ] B→A message with window unfocused: notification with text
      (preview on) / "New message" (preview off), Reply + Mark-read.
- [ ] Close window with close-to-tray: process stays, messages still notify.
- [ ] Autostart entry created when enabled, removed when disabled.
- [ ] Tray icon: DEFERRED — no tray host in CI/xvfb to verify SNI
      against; `notify.Tray` interface + `NoopTray` in place.

## WL-2.6 chat info + settings

- [ ] Drawer shows title/about; every settings row flips its key live.

## WL-2.7 search

- [ ] `go test ./internal/store/ -run FTS` green; global search finds
      week-old text; in-chat scope filters to the chat.

## WL-2.8 presence/typing/receipts/avatars

- [ ] B typing → "typing…" within 2 s; ticks sent→delivered→read.
- [ ] Avatars: initials fallback (photo fetch deferred with tray verdict).

## Gate 2

Daily-driver test: 3 days primary use by owner, no data loss, no
missed-message bug open. Then tag `whatsapp-linux-v0.1.0` (§16).
