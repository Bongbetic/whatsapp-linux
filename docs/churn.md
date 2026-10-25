# Dependency churn drill (WL-6.4)

Full interop after a pin bump must fit in one day. Procedure:

1. `go get go.mau.fi/whatsmeow@<new> github.com/diamondburned/gotk4/pkg@<new>`
   (one module at a time; gotk4 only after Void pango ≥ 1.57).
2. `go mod tidy && make vet && make test` — fix API drift in
   `internal/core` first (whatsmeow), then `internal/ui` (gotk4).
3. Re-run `LICENSES/` audit when libsignal/whatsmeow pins move.
4. Execute `tools/interop/phase1.md` two-account script, then the
   phase2 smoke (pair → send → history).
5. Commit as `chore(deps): bump <mod> <old>→<new>` with the interop
   result in the message. Roll back on any missed-message bug.

Evidence this repo pins safely today:

- whatsmeow `v0.0.0-20261017175727-b2b1e71d45e2` (136 Client
  methods / 68 events at audit).
- gotk4/pkg `v0.3.1`: `v0.4.1` fails against system pango 1.56.3
  (`pango_font_color_get_type`); re-test after pango ≥ 1.57.
- go-sqlite3 `v1.14.52` with `sqlite_fts5` tag (FTS5 proven in
  `TestSearchFTS`).
- Fuzz corpus anchors: `FuzzParseLink`, `FuzzScrub` (20 s runs,
  millions of execs, clean 2026-10-26).
