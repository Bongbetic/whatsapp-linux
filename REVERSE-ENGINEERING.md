# Reverse engineering evidence (moved from plan §1)

This file holds the analysis notes behind `whatsapp-linux.md` §1.
The reference artifacts themselves (`*.msix`, `*.MsixBundle`,
`*.Appx`, `*.exe`, `prs-response.json`, `AppxMetadata/`) live in
this directory on the analyst's machine only and are **never
committed** (see `.gitignore`). No Meta binaries, assets, fonts,
sounds, or decompiled code exist anywhere in the repo or package;
`whatsapp-linux` is a clean-room native client.

## Reference artifacts

| Artifact | SHA-256 |
|---|---|
| `WhatsApp Installer.exe` | `148e6ff8f2574568e0e902f890db29d72b476b0c2ecae917b4cb89638aa94d13` |
| `5319275A.WhatsAppDesktop_…MsixBundle` | `7a6920c14bcecf037dbe2a97ee45d20c54a9f6eb00c2743724fb29cb9235e110` |
| `WhatsApp.Root_2.2639.0.0_x64.msix` | inside bundle; PRS record in `prs-response.json` |

REA evidence IDs: `ev_cfcf7dae…` (installer),
`ev_072a482d…` (Root identity), `ev_55ffb4a6…` (Core identity),
`ev_dad1bf61…` (Core members unavailable — metadata exceeds REA
64 MB budget), `ev_9ac6a5fd…` (Rust lib). Root member inventory:
`/tmp/wa-re/root-members.json`.

## Established facts

- Installer = Microsoft StoreInstaller stub, product `9NKSQGP7F2NH`.
- Windows app = .NET 8 / WinUI 3 hybrid (`win_hybrid`), native
  Core (34,047 methods), native VoIP, Rust media lib, 12 SQLite
  stores, 95 XAML views dominated by call surfaces.
- Parity is judged on user-visible behavior and protocol interop,
  not on Meta's internal module split.
- Pinned whatsmeow exposes 136 `Client` methods and 68 event
  types (`types/events`), counted 2026-10-26.
- gotk4 `v0.3.1` builds, links, and runs against Void pango
  1.56.3 / gtk4 4.22.4 after installing
  `gobject-introspection` + `libgirepository-devel` (probe binary
  `/tmp/wa-impl/gtkprobe-bin`, "gotk4 links ok"). `v0.4.1` fails:
  generated code references `pango_font_color_get_type`
  (Pango ≥ 1.57), absent here. Pin `v0.3.1`; re-test newer gotk4
  only after Void pango ≥ 1.57.
