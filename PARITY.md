# Parity matrix — Windows Desktop 2.2639.0.0 vs whatsapp-linux

Executed 2026-10-26 at `91d3b57`+phase8 (WL-8.1). 1.0 gate: zero
silent gaps — every row below carries a verdict. Note: the plan
cites appendices §A–L that were never written; the sections below
reconstruct them from §7/§9/§12/§13 plus reference behavior, and
**this file is authoritative**.

Legend: `pass` (evidence this session) · `gap-documented` (1.0 may
ship; reason + follow-up named) · `open` (needs test SIM, owner
action, or fork — blocks its gate, not the audit).

Live two-account rows are `open` everywhere: they need the owner's
test SIM (plan §18 prerequisite), not more code. Code-complete
means implemented + unit/smoke-proven headless.

## A — pairing & connection (WL-0.1/1.1)

| Behavior | Ticket | Status |
|---|---|---|
| QR pair, phone-number pair, logout | WL-1.1 | open (code-complete; QR channel order fixed in xvfb smoke; live scan needs SIM) |
| reconnect banner offline/connecting/connected | WL-2.1 | open (code-complete; offline path needs live drop) |
| already-running single instance + URI handoff | WL-2.1 | gap-documented (URIs parse+validate, exit codes set; GApplication uniqueness free, handoff TODO WL-2.1b) |

## B — messaging (WL-1.4/2.3/2.4)

| Behavior | Ticket | Status |
|---|---|---|
| send/receive text, reply, edit, revoke, forward | WL-2.3/2.4 | open (code-complete; DTO conversion unit-tested; round-trip needs SIM) |
| reactions, stars, polls + votes | WL-2.4 | open (code-complete; needs SIM) |
| receipts ticks, presence/typing | WL-2.8 | open (code-complete; needs SIM) |
| history sync ingest | WL-1.3 | open (code-complete, monotonic progress; fixture sync needs SIM) |
| app-state sync (pin/mute/archive/star/read) | WL-1.5 | pass (typed events update app.db: `TestChatStateSync`; UI re-lists via RefreshEvent) |

## C — media (WL-3.x)

| Behavior | Ticket | Status |
|---|---|---|
| voice notes record/play/waveform | WL-3.1 | pass headless (OGG chain produces audio, 64-peak waveform) + open live (phone render needs SIM) |
| viewer, video notes, albums | WL-3.2 | open (code-complete; needs SIM) |
| downloads manager + auto-download rules | WL-3.3 | pass (queue cap + policy unit-tested) + open live |
| stickers/WebP + emoji | WL-3.4 | pass (convert + picker unit-tested) + open live; GIF-as-GifPlayback gap-documented (WL-3.4b) |
| image editor v1 | WL-3.5 | pass (crop/rotate/draw unit-tested); text gap-documented (WL-3.5b, `ErrTextUnsupported` hook) |
| original sounds | WL-3.6 | pass (synthesized WAVs verified valid; paplay hookup) |
| playback speed 1.5×/2× | WL-3.1 | gap-documented (no GtkMediaStream rate API; waits for Phase 5 player) |

## D — storage (§6/WL-3.3/6.2)

| Behavior | Ticket | Status |
|---|---|---|
| usage by chat, clear cache | WL-3.3 | pass (`TestStorage`, settings UI) |
| backup on upgrade + export chat (txt/json) | WL-6.2 | pass (`TestBackupAndExport`, `TestUpgradeAutoBackup`, wlprobe export proven) |

## E — groups (WL-4.1) / F — communities (WL-4.2)

| Behavior | Ticket | Status |
|---|---|---|
| create/edit/members/invites/approval queue | WL-4.1 | open (code-complete incl. dialog; approval manage read-only — gap-documented) |
| community view/create/link/announce | WL-4.2 | open (code-complete; needs SIM) |

## G — status (WL-4.3) / H — channels (WL-4.4)

| Behavior | Ticket | Status |
|---|---|---|
| view/post/privacy/viewers | WL-4.3 | open (code-complete; needs SIM) |
| follow/feed/react/search/post-if-eligible | WL-4.4 | open (code-complete; directory search + post-as-channel gap-documented, eligibility verdict here) |

## I — business (WL-4.5) / J — privacy & safety (WL-4.6)

| Behavior | Ticket | Status |
|---|---|---|
| profile/catalog/product message | WL-4.5 | open live (profile dialog code-complete); catalog/cart gap-documented (eligibility verdict here) |
| blocklist, safety numbers, report, disappearing | WL-4.6 | open live (page + engine code-complete) |

## K — calls (WL-5.x)

| Behavior | Ticket | Status |
|---|---|---|
| signaling matrix (offer/ring/terminate/reject → log) | WL-5.3 | pass (voip lifecycle unit-tested; upstream events wired) |
| 1:1 voice 60 s both directions (Gate 5a) | WL-5.2 | open (needs fork WL-5.1 + SIM) |
| 1:1 video both directions (Gate 5b) | WL-5.4 | open (needs fork + SIM) |
| group calls + links + screen share (Gate 5c) | WL-5.5/5.6 | open (needs fork + SIM); links report fork requirement honestly |

## L — platform (WL-2.5/6.x/7.x)

| Behavior | Ticket | Status |
|---|---|---|
| notifications + reply/mark-read, autostart | WL-2.5 | pass (policy unit-tested; autostart round-trip tested) + open live |
| tray icon | WL-2.5 | gap-documented (no tray host in CI/xvfb; `Tray` interface + `NoopTray` in place) |
| a11y + i18n | WL-6.3 | pass (tooltip pass; en catalog + fallback tested) |
| fuzz + race | WL-6.1 | pass (2 fuzzers clean; full suite `-race` green) |
| churn drill | WL-6.4 | pass (`docs/churn.md` + pin evidence) |
| Meta-AI + payments audit | WL-4.8 | pass (`docs/audit-ai-payments.md`: both out of v1) |
| xbps install → pair → round-trip, zero Meta files | WL-7.2 | pass install+launch+files (2026-10-26 podman run); pair/round-trip open (needs SIM) |
| release flow | WL-7.3 | pass dry-run (throwaway key); real publish needs owner OK |

## Gate 1.0 verdict

No silent gaps. Ship-blockers are all external: test-SIM live
runs (owner), whatsmeow fork place/accept (WL-5.1), and owner
release approvals. Documented gaps (tray, GIF flag, editor text,
speed, single-instance handoff, directory/catalog eligibility)
are each 1.0-shippable with the named follow-up.

## Re-verdict 2026-10-26 (`994d38c` + working tree, 28 files)

`go vet` clean, `gofmt` clean, `go test -tags sqlite_fts5 ./...`
green, `-race` green on store/core/logx/links/voip,
`make build` + xvfb `--version` ok. GUI booted headless under
xvfb (QR pairing flow live, engine on the fork), singleton
verified live (URI second-instance → exit 0, bare → exit 3),
zero GTK criticals.

| Change | Old row | New status |
|---|---|---|
| contact ingest (device-store sync, PushName/BusinessName/Contact events, history pushnames, DM title resolution + backfill) | names missing (live bug) | open live (code-complete; needs owner SIM to confirm) |
| chatinfo Mute → `Engine.SetMuted` (SendAppState) with toggle | log-only stub | open live (code-complete) |
| notification inline-reply → SendText; click → open-chat (was missing action) | log-only stub | open live (code-complete) |
| new-chat dialog → normalize + open/create DM | log-only stub | pass headless (no peer needed) |
| single-instance socket handoff (`internal/app/single.go`, exit 0/3) | gap-documented WL-2.1b | pass (verified live under xvfb) |
| StatusNotifier tray (`internal/notify/sni.go`, godbus) + NoopTray fallback; `SetConnected` wired | gap-documented | pass headless (fallback path in CI/xvfb) + open live (host tray) |
| chat-info drawer now populated on openChat (was never called) | silent gap | pass headless |
| shortcuts full §9 set (Ctrl+F / Ctrl+Shift+F in-chat find / Ctrl+Shift+M mute / Ctrl+Q tray / Esc) | partial | pass headless |
| in-chat find, chat-list focus/clear search | missing | pass headless |
| narrow <700px single-pane + Back, window size persist, CSS actually loaded (`loadCSS`), theme-aware bubbles, `style.css` installed to `/usr/share/whatsapp-linux/` | missing / dead code | pass headless (owner visual check on desktop) |
| fork `Bongbetic/whatsmeow` (offer/preaccept/accept/terminate) + `replace` in go.mod; engine CallPlace/CallAccept/CallHangup wired; transport relay parse → GStreamer audio leg; stanza debug capture; ringtone loop; call window mute/ended real | fork missing, ErrCallsForkRequired | open (code-complete; Gate 5a/5b need owner's live peer: place call → ring? → accept → audio?) |
| spell-check toggle stored but not consumed | silent gap | gap-documented (needs libspelling GIR follow-up, post-1.0) |

Still external: live SIM runs for every `open` row above, and
the owner's call peer test. `go.mod` gains `godbus/dbus` (pure
Go, no new system dep) and the fork `replace` (D-2).
