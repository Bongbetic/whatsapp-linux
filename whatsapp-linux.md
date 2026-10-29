# whatsapp-linux — implementation-ready plan

Status: ready to implement. This document is the single source of
truth for building `whatsapp-linux`: scope, pinned dependencies,
repo layout, package APIs, database schema, UI screens, tickets
with acceptance criteria, tests, packaging, and release.

Native Linux WhatsApp client, complete feature/function parity
with WhatsApp Desktop for Windows **2.2639.0.0**. Not a web-client
wrapper: the app speaks the WhatsApp multi-device companion
protocol natively and renders a GTK 4 UI. Clean-room: no Meta
binaries, assets, fonts, sounds, or decompiled code anywhere in
repo or package. Unofficial; not affiliated with WhatsApp LLC/Meta.

Conventions: ticket IDs `WL-<phase>.<n>`; decision IDs `D-<n>`;
all sizes relative (S <1 wk, M 2–4 wk, L 1–2 mo, XL 2–4 mo).
A ticket is done only when its acceptance criteria pass and
`go test ./...` is green. A phase is done only at its gate.

---

## 1. Evidence base (why the plan looks like this)

Reference artifacts in this directory (never committed):

| Artifact | SHA-256 |
|---|---|
| `WhatsApp Installer.exe` | `148e6ff8f2574568e0e902f890db29d72b476b0c2ecae917b4cb89638aa94d13` |
| `5319275A.WhatsAppDesktop_…MsixBundle` | `7a6920c14bcecf037dbe2a97ee45d20c54a9f6eb00c2743724fb29cb9235e110` |
| `WhatsApp.Root_2.2639.0.0_x64.msix` | inside bundle; PRS record in `prs-response.json` |

REA evidence:
`ev_cfcf7daeda22f388c7a1dc91eb53bfb3119e6b12fe85ad93b04e0f168cfe59fe` (installer),
`ev_072a482da7f1fbba22ecfbc4808b4685b39059aeaa779611d592a5ba10b493be` (Root identity),
`ev_55ffb4a6d3a4f956da7551fc17fa307330065cb624f9068155f05e9aa7831cbb` (Core identity),
`ev_dad1bf61211c65523d227e36077d4443181bcd557b6082c759862fa06116c8c2`
(Core members unavailable — metadata exceeds REA 64 MB budget),
`ev_9ac6a5fd390370abdeb0eab2cf4d0955af71ddf899db119e45f27929b441770b` (Rust lib).
Root member inventory: `/tmp/wa-re/root-members.json`.

Established facts (details in git history of this file):
installer = Microsoft StoreInstaller stub, product `9NKSQGP7F2NH`;
Windows app = .NET 8/WinUI 3 hybrid (`win_hybrid`), native Core
(34,047 methods), native VoIP, Rust media lib, 12 SQLite stores,
95 XAML views dominated by call surfaces. Parity is judged on
user-visible behavior and protocol interop, not Meta's split.

## 2. Decisions (locked unless a gate falsifies them)

| ID | Decision | Why / falsifier |
|---|---|---|
| D-1 | Language: **Go**, single CGO binary | whatsmeow is Go; Void `go-1.27.3_1`. Falsifier: gotk4 unworkable → D-3 fallback |
| D-2 | Protocol: **whatsmeow upstream first**; fork only when a ticket needs missing API (calls), via `replace` to `Bongbetic/whatsmeow` fork, upstream-merge weekly | 136 `Client` methods + 68 event types already cover most of §7; fork-everything rots |
| D-3 | UI: **GTK 4 + gotk4**, plain GTK first; libadwaita only if a working binding passes probe `WL-0.2` | Native GNOME; fallback if probe fails twice: Qt 6 via `therecipe/qt`-class binding is rejected (dead) → fallback is GTK via C shared lib called from Go (cgo shim), decided at Gate 0 |
| D-4 | Storage: **two SQLite files** — `session.db` owned by whatsmeow `sqlstore`, `app.db` owned by us (§6). Driver: `github.com/mattn/go-sqlite3` (CGO already required) | Session keys must never be touched by app migrations |
| D-5 | Media: **GStreamer** for playback/record/transcode/call media; PipeWire for devices; xdg-desktop-portal for screen share | Void `gstreamer1-1.28.6`, `pipewire-1.6.9` verified |
| D-6 | Account model: **one account per OS user** in v1 (multi-account = post-1.0) | Windows app is single-account |
| D-7 | Identity: package/binary `whatsapp-linux`, app ID `com.bongbetic.WhatsAppLinux`, display “WhatsApp (unofficial)” | Trademark-safe nominative use |
| D-8 | Telemetry: **none**. Crash reports local file only, opt-in attach | Unlike reference (StoreAgent/Asimov/UET) |
| D-9 | Secrets: noise/identity keys live only in whatsmeow store; no key logging ever; log scrubber for JIDs/phone numbers in debug logs (hash-prefix) | Ban + privacy hygiene |
| D-10 | Device identity: present as a normal desktop companion (whatsmeow defaults, `store.DeviceProps` set to a desktop Linux identity at `WL-1.1`); no automation/bulk APIs in UI | Minimize ban risk |

## 3. Dev environment (Void, exact)

```sh
sudo xbps-install -Sy go gtk4-devel libadwaita-devel \
  gobject-introspection libgirepository-devel pkg-config \
  gstreamer1-devel gst-plugins-base1-devel gst-plugins-good1 \
  gst-plugins-bad1 pipewire sqlite-devel libwebp-devel \
  xdg-desktop-portal xvfb-run git podman
go version          # expect go1.27.x
pkg-config --modversion gtk4 gstreamer-1.0  # 4.22.x / 1.28.x
```

Toolchain probes that must pass before Phase 1 (Gate 0):
`go build` of a gotk4 hello-window and of a whatsmeow QR-login
harness (tickets `WL-0.1`, `WL-0.2`). Known on 2026-10-26:
whatsmeow resolves to `v0.0.0-20261017175727-b2b1e71d45e2`.
gotk4 proven on this machine 2026-10-26: `v0.3.1` builds, links,
and runs against Void pango 1.56.3 / gtk4 4.22.4 after
`xbps-install gobject-introspection libgirepository-devel`
(probe binary `/tmp/wa-impl/gtkprobe-bin`, “gotk4 links ok”).
`v0.4.1` fails: generated code references
`pango_font_color_get_type` (Pango ≥1.57), absent here. Pin
`v0.3.1` (§4); re-test newer gotk4 only after Void pango ≥1.57.

## 4. Dependency pins (update only via a `chore(deps)` commit + interop pass)

| Module | Pin (initial) | Use | License* |
|---|---|---|---|
| `go.mau.fi/whatsmeow` | `v0.0.0-20261017175727-b2b1e71d45e2` | protocol engine (D-2) | MPL-2.0 (verified `LICENSE` 2026-10-26) |
| `go.mau.fi/libsignal` | via whatsmeow (`v0.2.2`) | Signal protocol | AGPL-3.0 (verify at `WL-1.6`) |
| `github.com/diamondburned/gotk4/pkg/{gtk/v4,gio/v2,glib/v2,gdk/v4}` | `v0.3.1` (proven §3) | GTK 4 bindings | MPL-2.0 (verify) |
| `github.com/mattn/go-sqlite3` | latest `v1.14.x` at `WL-1.2` | SQLite driver (both DBs) | MIT |
| `github.com/rs/zerolog` | via whatsmeow | logging (`waLog` adapter) | MIT |
| `rsc.io/qr` or `github.com/skip2/go-qrcode` | pick at `WL-0.1` (whichever renders to PNG bytes with no CGO) | pairing QR in CLI harness; UI draws its own from modules later | BSD/MIT |

*License column is provisional until the `WL-1.6` audit writes
`LICENSES/` with full texts. AGPL in the tree (libsignal via
whatsmeow) means the distributed binary's effective terms need
the audit's written verdict **before 0.1.0**; repo code stays MIT,
go module separation documented there. If the verdict blocks
distribution, Gate 1 must re-plan the signal layer — do not ship
first and audit later.

System (XBPS runtime) deps: `gtk4`, `gstreamer1`,
`gst-plugins-base1`, `gst-plugins-good1`, `gst-plugins-bad1`,
`pipewire`, `sqlite`, `libwebp`, `xdg-desktop-portal`,
`hicolor-icon-theme`, `desktop-file-utils`. Build deps add the
`-devel` set from §3.

## 5. Repo layout (create exactly this at `WL-1.0`)

```
whatsapp-linux/
  go.mod  go.sum  Makefile  README.md  PARITY.md
  LICENSE  LICENSES/            # third-party texts (WL-1.6)
  REVERSE-ENGINEERING.md        # evidence, moved from plan §1
  cmd/
    whatsapp-linux/main.go      # GUI entry; flags §11
    wlprobe/main.go             # headless protocol harness (test tool, not shipped)
  internal/
    app/        # Gtk.Application lifecycle, single instance, CLI flags
    bridge/     # DTOs + Engine/Store/Media interfaces (UI's only deps)
    core/       # whatsmeow wrapper: connection, events->DTO, send paths
    store/      # app.db: migrations/, repositories (chats, msgs, …)
    ui/         # GTK screens §9 (login/, chatlist/, convo/, settings/, call/, widgets/)
    media/      # GStreamer: play, record, transcode, waveform, thumbs
    voip/       # call state machine, signaling, pipelines (Phase 5)
    notify/     # GNotification + StatusNotifier tray + badge
    autostart/  # XDG autostart writer
    links/      # whatsapp://, tel:, wa.me parsing/rewriting
    settings/   # GSettings schema wrapper + defaults
    logx/       # zerolog setup + JID scrubber (D-9)
  data/
    com.bongbetic.WhatsAppLinux.desktop
    com.bongbetic.WhatsAppLinux.metainfo.xml
    com.bongbetic.WhatsAppLinux.gschema.xml
    icons/hicolor/{48x48,scalable}/apps/com.bongbetic.WhatsAppLinux.{png,svg}
    sounds/                     # original tones (WL-3.6), with CREDITS
  tools/
    build-xbps.sh               # §13
    interop/                    # manual test scripts + checklists per phase
  srcpkgs/whatsapp-linux/{template,update}
  .github/workflows/{ci.yml,release.yml}
  .gitignore                    # *.msix *.MsixBundle *.Appx *.exe /dist/ *.db
```

Rules: `internal/ui` imports only `bridge`, `settings`, `notify`,
`links` — never `core`/`whatsmeow`. `core` imports only
`bridge`(DTOs), `store`, whatsmeow. No package creates goroutines
that touch GTK except via `glib.IdleAdd` (see §8).

## 6. Data model

Paths (XDG): data `~/.local/share/whatsapp-linux/` holding
`session.db` (whatsmeow), `app.db`, `media/` (downloaded files by
`<chat-jid>/<msg-id>.<ext>`), `avatars/`, `stickers/`;
cache `~/.cache/whatsapp-linux/` (thumbnails, link previews);
config via GSettings + `~/.config/whatsapp-linux/` (autostart
marker, window state fallback).

`app.db` (SQLite, WAL, `foreign_keys=ON`). Migrations are
numbered SQL files in `internal/store/migrations/NNNN_*.sql`,
applied by a 40-line migrator tracking `PRAGMA user_version`;
never edit a shipped migration. Initial schema (0001):

```sql
CREATE TABLE chats(
  jid TEXT PRIMARY KEY,            -- user/group/newsletter JID string
  kind TEXT NOT NULL,              -- dm|group|community|channel|status-broadcast
  title TEXT NOT NULL DEFAULT '',
  avatar_path TEXT, unread INTEGER NOT NULL DEFAULT 0,
  mentions INTEGER NOT NULL DEFAULT 0,
  archived INTEGER NOT NULL DEFAULT 0, pinned INTEGER NOT NULL DEFAULT 0,
  muted_until INTEGER NOT NULL DEFAULT 0,  -- unix; -1 = always
  ephemeral_secs INTEGER NOT NULL DEFAULT 0,
  last_msg_id TEXT, last_ts INTEGER NOT NULL DEFAULT 0,
  draft TEXT NOT NULL DEFAULT '');
CREATE TABLE messages(
  chat_jid TEXT NOT NULL REFERENCES chats(jid),
  id TEXT NOT NULL,                -- stanza id
  sender_jid TEXT NOT NULL, is_from_me INTEGER NOT NULL,
  ts INTEGER NOT NULL,             -- unix seconds
  kind TEXT NOT NULL,              -- text|image|video|audio|voice|doc|sticker|
                                   -- location|live-location|contact|poll|event|
                                   -- reaction|system|call-log|revoked|edited
  body TEXT NOT NULL DEFAULT '',   -- text/caption (plaintext at rest; see note)
  quoted_id TEXT,                  -- replied-to message id
  media_path TEXT, media_mime TEXT, media_size INTEGER,
  media_sha256 BLOB, thumb_path TEXT, waveform BLOB,
  state TEXT NOT NULL,             -- sending|sent|delivered|read|played|failed
  edited INTEGER NOT NULL DEFAULT 0, starred INTEGER NOT NULL DEFAULT 0,
  view_once INTEGER NOT NULL DEFAULT 0,
  raw_proto BLOB,                  -- serialized waE2E.Message for reparse
  PRIMARY KEY(chat_jid,id));
CREATE INDEX idx_msg_chat_ts ON messages(chat_jid, ts);
CREATE TABLE reactions(
  chat_jid TEXT, msg_id TEXT, sender_jid TEXT, emoji TEXT, ts INTEGER,
  PRIMARY KEY(chat_jid,msg_id,sender_jid));
CREATE TABLE contacts(
  jid TEXT PRIMARY KEY, lid TEXT, phone TEXT,
  push_name TEXT NOT NULL DEFAULT '', contact_name TEXT NOT NULL DEFAULT '',
  about TEXT NOT NULL DEFAULT '', avatar_path TEXT,
  blocked INTEGER NOT NULL DEFAULT 0, is_business INTEGER NOT NULL DEFAULT 0);
CREATE TABLE group_members(
  group_jid TEXT, user_jid TEXT, role TEXT NOT NULL, -- member|admin|superadmin
  PRIMARY KEY(group_jid,user_jid));
CREATE TABLE polls(
  chat_jid TEXT, msg_id TEXT, question TEXT, options_json TEXT,
  selectable INTEGER, PRIMARY KEY(chat_jid,msg_id));
CREATE TABLE poll_votes(
  chat_jid TEXT, msg_id TEXT, voter_jid TEXT, options_json TEXT, ts INTEGER,
  PRIMARY KEY(chat_jid,msg_id,voter_jid));
CREATE TABLE calls(
  id TEXT PRIMARY KEY, chat_jid TEXT, direction TEXT, -- in|out
  kind TEXT,                     -- audio|video
  started_ts INTEGER, duration_s INTEGER, outcome TEXT); -- answered|missed|rejected|failed
CREATE TABLE downloads(
  chat_jid TEXT, msg_id TEXT, state TEXT, bytes_done INTEGER, bytes_total INTEGER,
  PRIMARY KEY(chat_jid,msg_id));
CREATE TABLE kv(key TEXT PRIMARY KEY, value TEXT); -- schema/meta only, not settings
```

Encryption-at-rest note: `body`/`raw_proto` are plaintext like
the reference `messages.db` (OS account is the trust boundary);
document in README, offer `sqlcipher` only as post-1.0 ticket.
Message search: FTS5 virtual table `messages_fts(body)` added in
migration 0002 with triggers (ticket `WL-2.7`), not in 0001.

Repositories (`internal/store`): `Chats`, `Messages`, `Contacts`,
`Groups`, `Polls`, `Calls`, `Downloads`, each with `Upsert/Get/List`
methods taking `context.Context`; all writes in transactions;
list queries are keyset-paginated (`ts`, `id`) — no OFFSET.

## 7. Protocol mapping (whatsmeow surface, measured 2026-10-26)

Pinned whatsmeow exposes **136 `Client` methods** and **68 event
types** (`types/events`). Core wrapper must map, at minimum:

| Feature | whatsmeow call(s) | Event(s) in |
|---|---|---|
| Pair QR / code | `GetQRChannel`, `PairPhone`, `Connect` | `QR`, `PairSuccess`, `PairError` |
| Connection | `Connect/Disconnect/Logout`, `WaitForConnection` | `Connected`, `Disconnected`, `LoggedOut`, `StreamReplaced`, `TemporaryBan`, `ClientOutdated` |
| Send text/media | `SendMessage`, `Upload/UploadReader`, builders `BuildReaction/BuildPollCreation/BuildEdit/BuildRevoke` | `Message`, `FBMessage`, `UndecryptableMessage` |
| Download | `Download`, `DownloadToFile`, `DownloadThumbnail`, `DownloadMediaWithPath(ToFile)` | `MediaRetry(Error)` |
| Receipts/presence | `MarkRead`, `SendChatPresence`, `SendPresence`, `SubscribePresence` | `Receipt`, `ChatPresence`, `Presence` |
| Groups | `CreateGroup`, `GetGroupInfo`, `UpdateGroupParticipants`, `SetGroupName/Topic/Photo/Announce/Locked/JoinApprovalMode`, `JoinGroupWithLink`, `GetGroupInviteLink`, `GetGroupRequestParticipants` | `GroupInfo`, `JoinedGroup` |
| Communities | `GetSubGroups`, `LinkGroup/UnlinkGroup`, `GetLinkedGroupsParticipants` | `GroupInfo` |
| App state | `FetchAppState`, `SendAppState` | `AppState`, plus typed `Pin/Archive/Mute/Star/MarkChatAsRead/DeleteChat/ClearChat/Blocklist/Contact/PushName…` |
| History sync | `DownloadHistorySync`, `BuildHistorySyncRequest` | `HistorySync`, `OfflineSyncPreview/Completed` |
| Privacy/block | `GetPrivacySettings`, `SetPrivacySetting`, `GetStatusPrivacy`, `GetBlocklist`, `UpdateBlocklist`, `SetDefaultDisappearingTimer`, `SetDisappearingTimer` | `PrivacySettings`, `Blocklist(Change)` |
| Status | `SetStatusMessage` + `SendMessage` to status broadcast | `Message` (status JID), `UserStatusMute` |
| Channels | `CreateNewsletter`, `Follow/UnfollowNewsletter`, `GetNewsletterInfo(WithInvite)`, `GetNewsletterMessages/GetNewsletterMessageUpdates`, `NewsletterMarkViewed`, `NewsletterSendReaction`, `NewsletterToggleMute`, `UploadNewsletter` | `NewsletterJoin/Leave/LiveUpdate/MuteChange/MessageMeta/Milestone` |
| Business | `GetBusinessProfile`, `GetOrderDetails`, `ResolveBusinessMessageLink`, `IsOnWhatsApp`, `GetUserInfo`, `GetProfilePictureInfo` | `BusinessName` |
| Stickers | `FetchStickerPack` + media upload | — |
| Calls (have) | `RejectCall` only; parsers emit offers | `CallOffer`, `CallOfferNotice`, `CallAccept/PreAccept/Reject/Terminate/Transport/RelayLatency`, `UnknownCallEvent` |
| Calls (**gap**) | place call, accept with media, group call, call link create/join | to implement in fork (Phase 5) |

Known gaps to close in fork (tickets in Phase 4/5): call
place/accept + media keys, call links, status posting parity
check, community create, channel directory search, catalog list
(beyond `GetOrderDetails`), message edit window errors, LID/PN
mapping edge cases (`StoreLIDPNMapping` exists), passkey pairing
(`PairPasskey*` events — expose only if phone offers it).

`internal/core.Engine` (the only API `bridge` sees):

```go
type Engine interface {
  Start(ctx) error; Stop() error
  Events() <-chan bridge.Event          // every DTO event, §8
  PairQR(ctx) (<-chan bridge.QRCode, error)
  PairPhone(ctx, number string) (code string, err error)
  Logout(ctx) error
  SendText(ctx, chat JID, body string, opts SendOpts) (MsgID, error)
  SendMedia(ctx, chat JID, f MediaFile, opts SendOpts) (MsgID, error)
  SendReaction/Edit/Revoke/Poll/Vote(ctx, …) error
  MarkRead(ctx, chat JID, ids ...MsgID) error
  SetPresence/Typing(ctx, …) error
  DownloadMedia(ctx, ref MediaRef, w io.Writer) error
  Group…(ctx, …)                       // 1:1 with table above
  Privacy…/Block…(ctx, …)
  // Phase 5:  CallPlace/CallAccept/CallReject/CallHangup(ctx,…)
}
```

DTOs (`internal/bridge`): `JID{User, Server, Device}`,
`Chat`, `Message{ID, Chat, Sender, Ts, Kind, Body, Media*, State,
QuotedID, Reactions[]}`, `Contact`, `GroupMeta`, `CallEvent`,
sealed `Event` sum (interface + `Kind()`), no protobuf types leak.

## 8. Runtime / threading

- Process = one `Gtk.Application` (`app`). GTK main loop owns UI.
- `core` runs whatsmeow on its goroutines; its event handler
  converts to DTO, writes store (worker pool of 2, ordered per
  chat by chat-JID hash), then posts `bridge.Event` on a buffered
  (256) channel. A single dispatcher goroutine forwards to GTK
  exclusively via `glib.IdleAdd`. Dropping policy: presence/typing
  events may be dropped at 90% buffer; messages never — block
  handler (whatsmeow tolerates slow handlers; log if >2 s).
- Sending is optimistic: `messages.state='sending'` row first, UI
  renders, engine confirms → `sent`/receipts update state.
- Downloads: max 3 concurrent, `downloads` table is the queue;
  auto-download policy from settings (§10) evaluated on insert.
- Shutdown order: stop accepting sends → `core.Stop()` (waits
  ≤5 s) → flush store WAL → quit GTK. SIGINT/SIGTERM same path.
- Reconnect: whatsmeow auto-reconnect + our backoff banner state
  (`Offline`/`Connecting`/`Connected`/`LoggedOut`) broadcast as
  `bridge.ConnEvent`; UI banner keys off it.

## 9. UI specification (GTK 4; every screen = 1 package under `internal/ui`)

Main window (`app`): `Gtk.ApplicationWindow` 1180×760 default
(min 860×560), header bar with connection state dot, two-pane
`Gtk.Paned` (left 340 px): chat list | conversation. Narrow
(<700 px) collapses to single pane with back button.

| Screen / package | Contents (widgets → behavior) |
|---|---|
| `ui/login` | QR canvas (modules drawn locally, refresh on `QR` event, countdown), “Link with phone number” toggle → entry + 8-char code display, status line, cancel |
| `ui/chatlist` | `Gtk.ListView` over `store.Chats` (avatar, title, last msg, ts, unread pill, mute/pin icons), search entry (filters + global FTS results section), archived row, context menu: archive/mute/pin/mark-read/delete; keyboard: ↑/↓, Enter opens |
| `ui/convo` | header (avatar, name, presence/“typing…”, call buttons 📞🎥, search, menu), `Gtk.ListView` of message rows (bubbles: text w/ formatting, media thumb, voice player, sticker, poll, location card, system lines), date separators, unread divider, reply banner, composer below |
| `ui/convo` composer | `Gtk.TextView` (Enter send / Shift+Enter newline per setting), attach menu (photo/video/file/camera later), emoji button, mic ⇄ send swap, recording bar (timer, waveform, cancel), reply/edit preview strip, typing debounce 3 s |
| `ui/widgets` | `Avatar` (initials fallback), `MessageRow`, `MediaThumb`, `VoicePlayer` (play/pause, seek, 1×/1.5×/2×), `PollView`, `ReactionBar`, `EmojiPicker` (grid + skin-tone popover + search, from system emoji data), `LinkPreview`, `Banner` (offline/ephemeral), `SpinnerRow` |
| `ui/chatinfo` | right drawer/page: photo, about, media-links-docs tabs, members list (groups: add/remove/promote, invite link + QR, approval queue), mute/ephemeral/block/report/exit, encryption row → safety numbers dialog |
| `ui/settings` | pages §10, `Gtk.Stack` + sidebar; every row binds one GSettings key or one privacy engine call (no local shadow state) |
| `ui/status` | own status row + contacts' rings list, viewer (image/video with progress bars, reply box, viewers sheet), post flow (text status composer + media picker) |
| `ui/channels` | followed list, channel page (updates feed, react, share), explore/search (Phase 4) |
| `ui/calls` | call log list (from `calls` table), new-call picker, call-link create/copy |
| `ui/call` | call window: video grid (`Gtk.Picture` per participant from GStreamer sinks), self PiP, controls bar (mute, camera, speaker, screen share, reactions, hang up), incoming-call dialog (accept/reject, ringtone), minimized floating bar, devices popover, waiting-room sheet (Phase 5) |
| `ui/mediaview` | fullscreen viewer: image zoom, video controls (GStreamer playbin), caption, prev/next in chat album, download/open-with |
| dialogs | new chat (contact picker + “message a number”), new group (member picker → name/photo), forward picker, delete/revoke confirm, logout, about |

Shortcuts (global, `Gtk.ShortcutController`): Ctrl+N new chat,
Ctrl+F chat search, Ctrl+Shift+F in-chat search, Ctrl+, settings,
Ctrl+=/-/0 zoom, Ctrl+Q quit-to-tray,
Ctrl+Shift+M mute chat, Esc back/close search. Accessibility:
all icon buttons get labels/tooltips; focus order chatlist →
convo → composer; high-contrast follows GTK theme. Theming:
system light/dark only in v1; WhatsApp-green accent via CSS
provider (`internal/ui/theme/style.css`, embedded in the binary),
no hard-coded colors in Go.

## 10. Settings (GSettings schema `com.bongbetic.WhatsAppLinux`)

Local keys (type, default): `autostart(b,true)`,
`close-to-tray(b,true)`, `start-hidden(b,false)`,
`enter-to-send(b,true)`, `font-scale(d,1.0)`,
`download-dir(s,"")` (= XDG Downloads), `auto-download-photos/
audio/video/docs(b,true/true/false/false)`,
`notify-preview(b,true)`, `notify-sound(b,true)`,
`spellcheck(b,true)`, `low-data-calls(b,false)`,
`window-w/h/x/y(i,…)` internal. Server-side (privacy) settings
are **not** GSettings: read/write through Engine
(`GetPrivacySettings`/`SetPrivacySetting`) and cache read-only
in `kv` for offline display, marked stale offline.

## 11. CLI surface

`whatsapp-linux [--hidden] [--reset-profile] [--version]
[--debug] [whatsapp://…|https://wa.me/…]`
`--reset-profile` deletes `app.db`+media but keeps `session.db`
(still logged in); `--reset-profile --full` also logs out and
deletes `session.db` (confirm prompt on tty / dialog in GUI).
Exit codes: 0 ok, 2 bad args, 3 already-running (single-instance
hands off and exits 0 when a URI was passed, else 3 — mirror
StoreInstaller `AlreadyRunning` semantics loosely), 4 store
locked/corrupt with message. `wlprobe` (dev only):
`wlprobe pair|send <jid> <text>|dump-events|call-probe` against
a throwaway profile dir env `WLPROBE_HOME`.

## 12. Media specification (`internal/media`, GStreamer)

- Playback: one reusable `playbin` wrapper (audio + video);
  voice notes are OGG/Opus (`.ogg`, mime `audio/ogg; codecs=opus`).
- Recording voice: PipeWire source → `opusenc` (16 kHz mono,
  32 kbps) → OGG; waveform = 64 normalized peaks computed during
  record, stored in `messages.waveform`; max length follows
  server limits discovered at `WL-3.1` (record in code const).
- Images: send as JPEG (quality 80, max edge 1600 px) unless
  “send as document”; thumbnails 320 px JPEG in cache.
- Video: probe with `gst-discoverer` API; transcode to MP4
  (H.264+AAC) only when source is not already MP4/H.264;
  streaming-sidecar fields from whatsmeow upload response.
- Stickers: WebP 512×512 (pad/cover), animated preserved if
  source already animated WebP; packs via `FetchStickerPack`.
- All pipelines are string constants in `media/pipelines.go`
  with unit tests asserting `gst_parse_launch` succeeds headless.

## 13. Calls specification (Phase 5, `internal/voip`)

State machine per call ID: `idle → offering|ringing →
preaccept → connecting(relay) → in-call → terminating → ended`,
driven by fork events (`CallOffer/Accept/PreAccept/Transport/
Reject/Terminate`). Fork work in whatsmeow `call.go` successor:
(a) serialize `<call>` children for offer/accept (enc keys from
libsignal session, relay list from `CallTransport`), (b) expose
`PlaceCall/AcceptCall` with opaque media params, (c) call-link
create/join nodes. Media: GStreamer `webrtcbin` is **not**
assumed compatible; plan A = raw RTP/SRTP elements
(`rtpopuspay`, `rtpvp8pay`/`rtph264pay`, `srtpenc` with keys
from signaling, `udpsink` to relay endpoints); plan B =
`webrtcbin` if interop shows DTLS-SRTP + ICE is what relays
expect. Decision by probe `WL-5.0` packet capture of own calls.
Echo cancellation/noise: PipeWire echo-cancel module if present,
else `webrtcdsp` (gst-plugins-bad). Screen share:
`xdg-desktop-portal` ScreenCast → PipeWire node → encode as
second video track. Ringing uses `notify` + sound; incoming
while locked screen = notification only (documented limit).

## 14. Tickets

### Phase 0 — probes & gates [M]

- **WL-0.1 whatsmeow pairing harness** (`/tmp` throwaway):
  QR login with test SIM, send text to self + phone.
  Accept: terminal shows QR, `PairSuccess`, echo round-trip <5 s;
  notes committed to `tools/interop/phase0.md` after repo exists.
- **WL-0.2 gotk4 pin probe** — DONE 2026-10-26: `v0.3.1` builds,
  links, runs (§3). Remaining accept item: real window under
  `xvfb-run` for 3 s in CI smoke (covered by WL-1.0).
- **WL-0.3 event-coverage run**: harness logs every event type
  while tester performs scripted phone actions (text, image,
  voice, reaction, poll, group create, status post, channel
  follow, call ring+hangup-unanswered).
  Accept: coverage table in `phase0.md` = each action → event(s)
  observed or “none”; call-offer node dump (own traffic) saved.
  **Gate 0** = WL-0.1..0.3 accepted + call-feasibility verdict
  (go / go-with-fork / blocked-with-evidence) written.

### Phase 1 — core + stores [L]

- **WL-1.0 repo scaffold** per §5 + CI (§15) + `.gitignore`.
  Accept: `make` builds empty GUI that opens a window.
- **WL-1.1 core engine skeleton**: sqlstore open
  (`sqlite3` DSN `file:session.db?_foreign_keys=on&_journal_mode=WAL`),
  `store.DeviceProps` desktop identity, connect/pair/logout,
  event→DTO for connection set. Accept: `wlprobe pair` works
  from in-repo code; unit test for DTO conversion (recorded JSON).
- **WL-1.2 app.db + migrator + repos** (§6, migration 0001).
  Accept: tests create DB in `t.TempDir`, migrate twice = noop,
  CRUD round-trips per repo, `-race` clean.
- **WL-1.3 history sync ingest**: `HistorySync` →
  chats/messages/contacts upserts, progress DTO.
  Accept: fixture sync (recorded, redacted) yields expected row
  counts; progress events monotonic 0→100.
- **WL-1.4 live message ingest + send**: text + media download/
  upload, receipts, optimistic states (§8).
  Accept: two-account manual interop script passes (text, image,
  doc both directions; states reach read).
- **WL-1.5 app-state sync**: pin/mute/archive/star/read reflected
  both ways. Accept: phone archives → CLI dump shows it; reverse.
- **WL-1.6 license audit** per §4 → `LICENSES/`, `NOTICE`.
  Accept: written distribution verdict signed off in PR text.
  **Gate 1** = WL-1.1..1.6 accepted.

### Phase 2 — messaging UI [XL] → 0.1.0

- **WL-2.1 app shell + login** (§9 `app`, `ui/login`), conn banner.
- **WL-2.2 chat list + convo read path** (text/media thumbs/system).
- **WL-2.3 composer send path** (text, formatting render + input
  shortcuts, reply, attachments, drafts saved on blur/500 ms).
- **WL-2.4 message actions**: react, edit, revoke, forward, star,
  multi-select, context menus; polls create/vote/results.
- **WL-2.5 notifications + tray** (`notify`): GNotification with
  Reply/Mark-read, SNI tray w/ unread, close-to-tray, autostart.
- **WL-2.6 chat info (DM) + settings skeleton** (§10 local keys).
- **WL-2.7 search**: migration 0002 FTS5, global + in-chat UI.
- **WL-2.8 presence/typing/receipt ticks**, avatar fetch/cache.
  Accept (each): demo checklist in `tools/interop/phase2.md`
  executed against phone; screenshots optional. **Gate 2** =
  daily-driver test: 3 days primary use by owner, no data loss,
  no missed-message bug open. Tag `whatsapp-linux-v0.1.0` via §16.

### Phase 3 — media complete [L]

- **WL-3.1 voice notes** record/play/waveform/speed (§12).
- **WL-3.2 media viewer + video notes + albums**.
- **WL-3.3 downloads manager** (queue UI, auto-download rules,
  storage page: usage by chat, clear cache).
- **WL-3.4 stickers/GIF/emoji**: picker (§9), pack view, sticker
  send from image (convert), GIF send as `GifPlayback` video.
- **WL-3.5 image editor v1**: crop/rotate/text/draw on send path.
- **WL-3.6 original sounds** (synthesized/licensed alerts +
  ringtone, `data/sounds/CREDITS`), per-chat tone picker.
  **Gate 3** = checklist §D rows pass interop.

### Phase 4 — social complete [XL]

- **WL-4.1 groups mgmt** (create/edit/members/invites/approval).
- **WL-4.2 communities** (view/create/link, announcement send).
- **WL-4.3 status** view/post/privacy/viewers.
- **WL-4.4 channels** follow/feed/react/search/post-if-eligible.
- **WL-4.5 business** profile/catalog/product msg/cart send.
- **WL-4.6 privacy & safety** settings pages, blocklist UI,
  safety-number dialog, report flow, disappearing defaults.
- **WL-4.7 fork gaps roundup** (§7 gaps minus calls).
- **WL-4.8 Meta-AI + payments audit** (written; scope verdict).
  **Gate 4** = §E–G, I, J, L interop rows pass.

### Phase 5 — calls [XL] (WL-5.0 starts in Phase 2)

- **WL-5.0 signaling probe**: capture own offer/accept nodes
  (phone↔Windows), relay/ICE evidence → media plan A/B decision.
- **WL-5.1 fork: PlaceCall/AcceptCall** + enc/transport plumbing.
- **WL-5.2 audio media path** (§13) — **Gate 5a**: 1:1 voice
  phone↔Linux 60 s, both directions audible, hangup clean.
- **WL-5.3 call UI** (§9 `ui/call`) + call log writes.
- **WL-5.4 video** — **Gate 5b**: 1:1 video both directions.
- **WL-5.5 group calls + call links + waiting room**.
- **WL-5.6 screen share** (portal) — **Gate 5c** with 5.5.
- **WL-5.7 devices/echo/low-data**, H2 ML effects = stretch.

### Phase 6 — hardening [M]

- **WL-6.1** fuzz decoders + `-race` CI; **WL-6.2** backup/export
  chat (txt/json) + store backup on upgrade; **WL-6.3** a11y +
  i18n (gettext, `en` source); **WL-6.4** churn drill: bump
  whatsmeow pin, full interop script ≤1 day.

### Phase 7 — packaging [M] (§16)

- **WL-7.1** `tools/build-xbps.sh` + template; **WL-7.2** podman
  install test; **WL-7.3** release workflow dry-run on `v0.0.0`
  tag in a private test repo or workflow_dispatch.
  **Gate 7** = §17 package checks pass.

### Phase 8 — parity audit [M]

- **WL-8.1** execute `PARITY.md` (every §A–L row, dated, build
  hash); **WL-8.2** fix-or-document loop. **1.0 gate**: zero
  silent gaps.

## 15. CI (`.github/workflows/ci.yml`)

Container `ghcr.io/void-linux/void-glibc`: install §3 build deps,
`go vet ./...`, `go test ./...` (unit + `xvfb-run` UI smoke),
`gofmt -l` empty, build binary artifact. No secrets, no real
accounts, no network beyond Go proxy. Runs on PR + `main`.

## 16. Packaging & release

Version: `VERSION` file == git tag `whatsapp-linux-vX.Y.Z`
(workflow fails on mismatch). `tools/build-xbps.sh`:

1. `CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -X main.version=$V" ./cmd/whatsapp-linux`
2. stage tree (§5 `data/` → `/usr/share/…`, binary → `/usr/bin/`)
3. `glib-compile-schemas` check, `desktop-file-validate`.
4. `xbps-create -A <arch> -n whatsapp-linux-${V}_1 -s "Unofficial WhatsApp client for Linux" --license "MIT, MPL-2.0, AGPL-3.0" (final per WL-1.6) --depends "gtk4>=4.20 gstreamer1 gst-plugins-base1 gst-plugins-good1 gst-plugins-bad1 pipewire sqlite libwebp xdg-desktop-portal" stage/`

`release.yml` (taema flow): per-arch matrix, x86_64 first
(extend only on green builds): install deps → `go test` →
build → `XBPS_ARCH=<a> xbps-rindex -a *.xbps` → write
`$XBPS_PRIVKEY` secret to a file, sign `--privkey <file>
--signedby "Bongbetic <soubarna@live.in>"` (`-s` repo, `-S`
package) → attach `.xbps`, `.sig2`, `<arch>-repodata`
→ post-release commit bumps `srcpkgs/whatsapp-linux/template`
+ README. Repo URL
`https://github.com/Bongbetic/whatsapp-linux/releases/latest/download`;
user install + fingerprint (`fa:d1:00:32:ce:b1:b9:fd:73:05:05:76:75:54:37:25`)
documented in README. XBPS ref:
https://docs.voidlinux.org/xbps/index.html. Repo creation,
secret upload, every push/tag: owner's explicit OK each time.

## 17. Validation matrix

| Layer | Command / action | Pass bar |
|---|---|---|
| unit | `go test ./...` | green, `-race` on store/core |
| UI smoke | `xvfb-run ./whatsapp-linux --version` + 3 s run | exit 0, profile dir made |
| fixtures | `go test ./internal/core -run Fixture` | recorded protobuf → DTO goldens |
| interop | `tools/interop/phaseN.md` two-account script | every row ticked, dated |
| package | podman Void: add repo, `xbps-install -S whatsapp-linux`, launch, pair | message round-trip; `xbps-query -Rf` has zero Meta files |
| churn | bump pin (§4) → full interop | ≤1 day, notes in `PARITY.md` |

## 18. First sprint (start immediately after `go`)

Day 1: WL-0.1 harness + test-SIM pairing; WL-0.2 pin probe.
Day 2–3: WL-0.3 coverage run; write `phase0.md`; Gate 0 verdict.
Day 4: WL-1.0 scaffold + CI. Day 5: WL-1.1 skeleton pairing
in-repo. Week 2: WL-1.2 schema/repos; WL-1.6 audit draft.
Week 3–4: WL-1.3/1.4. Gate 1 review at end of week 4.

Prerequisites from owner before Day 1: (1) test phone number
that may be banned without loss, (2) Windows reference machine
or VM with the analyzed build for side-by-side audits,
(3) `go` reply to this plan. Nothing else blocks.

## 19. Risks (top 5, owner-visible)

1. **Ban risk** (unsupported companion): mitigations D-9/D-10;
   test numbers for dev; README warning from 0.1.
2. **Calls may be server-gated**: WL-5.0 decides with evidence;
   messaging releases never blocked by it, parity claim is.
3. **Protocol churn**: pin + weekly upstream watch; WL-6.4 drill.
4. **gotk4 churn** (young bindings; pango pin incident §3):
   UI behind `bridge`, widget count kept small, regen path known.
5. **License audit outcome** (libsignal AGPL via whatsmeow):
   WL-1.6 gate before any public binary.

## 20. Sources

- https://pkg.go.dev/go.mau.fi/whatsmeow
- https://raw.githubusercontent.com/tulir/whatsmeow/main/README.md
- https://github.com/diamondburned/gotk4
- https://gstreamer.freedesktop.org/documentation/
- https://docs.voidlinux.org/xbps/index.html
- Local (2026-10-26): whatsmeow source at pinned version in Go
  module cache (136 methods / 68 events counted), gotk4 tag list
  via `git ls-remote`, `prs-response.json`, REA evidence §1,
  `xbps-query`/`pkg-config` on this Void x86_64 machine.

