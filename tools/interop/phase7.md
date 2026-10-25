# Phase 7 interop notes — Gate 7 (WL-7.1..WL-7.3)

## WL-7.1 build script + template

- [x] `bash tools/build-xbps.sh x86_64` produces
      `dist/whatsapp-linux-<V>_1.x86_64.xbps` on this Void host.
- [x] `glib-compile-schemas --strict --dry-run` passes on the
      staged schema; `desktop-file-validate` passes on the staged
      desktop entry.
- [x] `srcpkgs/whatsapp-linux/{template,update}` present;
      version+checksum bumped by the post-release job only.

## WL-7.2 podman install test

Run from the repo root (uses podman, never Docker):

```sh
podman run --rm -v ./dist:/dist:ro \
  ghcr.io/void-linux/void-glibc:latest sh -c '
    echo "repository=/dist" > /etc/xbps.d/90-test.conf &&
    xbps-install -S --repository=/dist whatsapp-linux &&
    xbps-query -Rf whatsapp-linux | grep -ci meta; :'
```

Pass bar: installs, launches (`--version` under xvfb when a
display harness exists), message round-trip against the test SIM,
and `xbps-query -Rf` lists zero Meta files. Record the dated run
below.

Runs:

- 2026-10-26 (`f5dc477`+phase7, `0.0.0.dev` test build): podman
  Void installs 235 pkgs, `--version` prints, file list has zero
  capital-M `Meta` hits. Message round-trip pending test SIM.
  Note: deps must all carry version floors — bare rundep names
  fail resolution in the container xbps (0.60.7); `libgirepository`
  is a runtime dep (gotk4 dlopens it).

## WL-7.3 release dry-run (2026-10-26)

- [x] Tag/version check passes for `whatsapp-linux-v0.0.0.dev`.
- [x] `tools/build-xbps.sh x86_64` green locally.
- [x] `xbps-rindex -a`, repo sign + package sign with a THROWAWAY
      key on a temp copy (note: every `xbps-rindex` call needing
      the key takes its own `--privkey`).
- [ ] Attach to a draft release in a private test repo or via
      `workflow_dispatch` — owner action.
- [ ] Real tags/releases need the owner's explicit OK every time
      (§16); nothing here pushes or publishes.
