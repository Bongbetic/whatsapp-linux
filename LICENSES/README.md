# Third-party licenses (WL-1.6 audit)

Repo's own code: MIT (`LICENSE`). Distributed binary's effective
terms per module (pins in `go.mod`):

| Module | Version | License | Text |
|---|---|---|---|
| `go.mau.fi/whatsmeow` | `v0.0.0-20261017175727-b2b1e71d45e2` | MPL-2.0 | `MPL-2.0.txt` (copy of upstream `LICENSE`) |
| `go.mau.fi/libsignal` | `v0.2.2` (via whatsmeow) | **GPL-3.0-only** (see below) | `libsignal-LICENSE` (copy of upstream `LICENSE`) |
| `github.com/diamondburned/gotk4/pkg` | `v0.3.1` | MPL-2.0 | `MPL-2.0.txt` (identical text) |
| `github.com/mattn/go-sqlite3` | `v1.14.52` | MIT | upstream `LICENSE` (short; notice kept in binary `--help` colophon — TODO WL-7.1) |
| `github.com/rs/zerolog` | `v1.35.1` | MIT | same as above |
| `github.com/skip2/go-qrcode` | see `go.mod` | MIT | same as above |

## Distribution verdict (signed off in Phase 1)

**GO with conditions.** Findings:

1. libsignal: the plan presumed AGPL-3.0, but the `LICENSE` file
   shipped in `v0.2.2` is the **GPL-3.0** text (674 lines, §13 is
   the GPLv3→AGPLv3 compatibility clause, no remote-network
   clause). Upstream may still declare AGPL-3.0 elsewhere, so the
   audit treats the **stricter reading (AGPL-3.0) as governing**
   until the pin is re-audited. Either way the obligation is the
   same for us: every public release must ship complete
   corresponding source (public git repo + release artifacts) and
   these license texts. No re-plan of the signal layer needed.
2. MPL-2.0 (whatsmeow, gotk4) is file-level copyleft: our MIT
   files stay MIT; binary distribution needs the MPL notice,
   satisfied by this directory + `NOTICE` (TODO WL-7.1).
3. Re-audit at every `chore(deps)` pin bump (WL-6.4 drill
   includes it). If a future libsignal pin ships a `LICENSE` that
   blocks distribution, Gate 1 re-plans the signal layer — do not
   ship first and audit later.

`NOTICE` (binary attribution colophon) is generated at WL-7.1.
