# Phase 1 interop notes — Gate 1 (WL-1.1..WL-1.6)

Two-account script (A = Linux build, B = phone). Throwaway profile
for A: `export WLPROBE_HOME=$(mktemp -d)`.

## WL-1.1 pair

- [ ] `go run ./cmd/wlprobe pair` prints ASCII QR, phone scans,
      `pair success: true`.
- [ ] Restart: `Start` with stored device connects silently
      (no QR), `ConnEvent{connected}` observed in `dump-events`.

## WL-1.3 history sync

- [ ] `dump-events` shows `sync-progress` monotonic 0→100.
- [ ] `app.db` chats/messages row counts match phone's visible
      history window (± live messages arriving during sync).

## WL-1.4 live message ingest + send

- [ ] A→B text, B→A text; states reach `read` both directions.
- [ ] A→B image, B→A image + doc; files land under
      `media/<chat-jid>/`, thumbnails cached.
- [ ] Kill -9 A mid-chat, restart: no message loss, no dupes
      (idempotent upserts on `(chat_jid,id)`).

## WL-1.5 app-state sync

- [ ] B archives a chat → `dump-events` shows change; A CLI dump
      reflects `archived=1`.
- [ ] Reverse: mutate via Engine, B reflects within 30 s.

## WL-1.2 / WL-1.6

- [ ] `go test -race ./internal/store/... ./internal/core/...`
      green (CI runs this).
- [ ] License audit: `LICENSES/README.md` verdict signed off.
