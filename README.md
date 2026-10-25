# whatsapp-linux

Unofficial native Linux WhatsApp client (GTK 4 + Go). Speaks the
WhatsApp multi-device companion protocol directly via
[whatsmeow](https://pkg.go.dev/go.mau.fi/whatsmeow) — not a
web-client wrapper. Clean-room: no Meta binaries, assets, fonts,
sounds, or decompiled code anywhere in the repo or package.

> **Unofficial; not affiliated with WhatsApp LLC / Meta.**
> Companion clients outside Meta's apps carry a ban risk: use a
> secondary/test number for daily driving until 1.0. See
> `REVERSE-ENGINEERING.md` for the analysis behind this project.

## Status

Built phase-by-phase from `whatsapp-linux.md` (plan; not shipped):

- [x] Phase 0 — probes & Gate 0 (`tools/interop/phase0.md`)
- [x] Phase 1 — core engine + stores + license audit
- [x] Phase 2 — messaging UI (0.1.0 gate: owner daily-driver test)
- [x] Phase 3 — media complete
- [x] Phase 4 — social complete
- [ ] Phase 5 — calls (signaling done; media gates need live verification)
- [ ] Phase 6 — hardening
- [ ] Phase 3 — media complete
- [ ] Phase 4 — social complete
- [ ] Phase 5 — calls
- [ ] Phase 6 — hardening
- [ ] Phase 7 — packaging
- [ ] Phase 8 — parity audit (`PARITY.md`)

## Build (Void Linux)

```sh
sudo xbps-install -Sy go gtk4-devel libadwaita-devel \
  gobject-introspection libgirepository-devel pkg-config \
  gstreamer1-devel gst-plugins-base1-devel gst-plugins-good1 \
  gst-plugins-bad1 pipewire sqlite-devel libwebp-devel \
  xdg-desktop-portal xvfb-run git
make build        # ./whatsapp-linux
make test         # full suite (adds -tags sqlite_fts5 for FTS5)
make smoke        # xvfb-run ./whatsapp-linux --version
```

Pinned: whatsmeow `v0.0.0-20261017175727-b2b1e71d45e2`,
gotk4/pkg `v0.3.1` (do not bump to v0.4.x until Void pango
≥ 1.57). Update only via a `chore(deps)` commit + interop pass.

## Run

```sh
./whatsapp-linux [--hidden] [--reset-profile [--full]] [--version] [--debug] [link]
```

First run shows a QR code; link it from your phone
(WhatsApp → Linked devices). Profile lives in
`~/.local/share/whatsapp-linux/` (`session.db`, `app.db`,
`media/`). Messages are stored **plaintext at rest**, like the
reference client's `messages.db` — your OS account is the trust
boundary. No telemetry, ever; crash logs stay local.

## Layout

`cmd/whatsapp-linux` (GUI) · `cmd/wlprobe` (headless harness,
dev only) · `internal/{app,bridge,core,store,ui,media,voip,
notify,autostart,links,settings,logx}` · `data/` (desktop,
metainfo, GSettings, icons) · `tools/interop/` (manual test
scripts) · `srcpkgs/` (XBPS template).

## License

Repo code: MIT (`LICENSE`). Binary effective terms:
`LICENSES/` (MPL-2.0 + libsignal copyleft; verdict: distribute
only with corresponding source — every release does).
