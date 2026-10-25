# Parity matrix — Windows Desktop 2.2639.0.0 vs whatsapp-linux

Executed at Phase 8 (WL-8.1). Until then this file is the working
matrix: each row names the user-visible behavior, the ticket that
implements it, and the interop proof. Note: the plan cites
appendices §A–L that were never written; the sections below
reconstruct them from §7/§9/§12/§13 plus reference behavior, and
**this file is authoritative** for the 1.0 gate (zero silent gaps).

Legend: `pass` (dated, build hash) · `gap-documented` (1.0 may
ship) · `open` (blocks 1.0).

## A — pairing & connection (WL-0.1/1.1)

| Behavior | Ticket | Status |
|---|---|---|
| QR pair, phone-number pair, logout | WL-1.1 | open |
| reconnect banner offline/connecting/connected | WL-2.1 | open |
| already-running single instance + URI handoff | WL-2.1 | open |

## B — messaging (WL-1.4/2.3/2.4)

| Behavior | Ticket | Status |
|---|---|---|
| send/receive text, reply, edit, revoke, forward | WL-2.3/2.4 | open |
| reactions, stars, polls + votes | WL-2.4 | open |
| receipts ticks, presence/typing | WL-2.8 | open |
| history sync ingest | WL-1.3 | open |
| app-state sync (pin/mute/archive/star/read) | WL-1.5 | open |

## C — media (WL-3.x)

| Behavior | Ticket | Status |
|---|---|---|
| voice notes record/play/waveform/speed | WL-3.1 | open |
| viewer, video notes, albums | WL-3.2 | open |
| downloads manager + auto-download rules | WL-3.3 | open |
| stickers/GIF/emoji | WL-3.4 | open |
| image editor v1 | WL-3.5 | open |
| original sounds | WL-3.6 | open |

## D — storage (§6/WL-3.3)

| Behavior | Ticket | Status |
|---|---|---|
| usage by chat, clear cache | WL-3.3 | open |
| backup/export chat (txt/json) | WL-6.2 | open |

## E — groups (WL-4.1) / F — communities (WL-4.2)

| Behavior | Ticket | Status |
|---|---|---|
| create/edit/members/invites/approval queue | WL-4.1 | open |
| community view/create/link/announce | WL-4.2 | open |

## G — status (WL-4.3) / H — channels (WL-4.4)

| Behavior | Ticket | Status |
|---|---|---|
| view/post/privacy/viewers | WL-4.3 | open |
| follow/feed/react/search/post-if-eligible | WL-4.4 | open |

## I — business (WL-4.5) / J — privacy & safety (WL-4.6)

| Behavior | Ticket | Status |
|---|---|---|
| profile/catalog/product message | WL-4.5 | open |
| blocklist, safety numbers, report, disappearing | WL-4.6 | open |

## K — calls (WL-5.x)

| Behavior | Ticket | Status |
|---|---|---|
| 1:1 voice 60 s both directions (Gate 5a) | WL-5.2 | open |
| 1:1 video both directions (Gate 5b) | WL-5.4 | open |
| group calls + links + screen share (Gate 5c) | WL-5.5/5.6 | open |

## L — platform (WL-2.5/6.x/7.x)

| Behavior | Ticket | Status |
|---|---|---|
| notifications + reply/mark-read, tray, autostart | WL-2.5 | open |
| a11y + i18n | WL-6.3 | open |
| xbps install → pair → round-trip, zero Meta files | WL-7.2 | open |
