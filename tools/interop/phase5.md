# Phase 5 interop notes — Gates 5a/5b/5c (WL-5.0..WL-5.7)

## WL-5.0 signaling probe (media plan A/B decision)

- [ ] `WLPROBE_HOME=$(mktemp -d) go run ./cmd/wlprobe call-probe`
      while the phone rings this device: `calls.log` shows
      offer → (preaccept) → terminate with IDs and relay hosts.
- [ ] Decision: plan A (raw RTP/SRTP, `voip` pipeline consts) vs
      plan B (webrtcbin) from relay/ICE evidence. **Pending live
      capture — default stays plan A.**

## WL-5.1 fork: PlaceCall/AcceptCall

- [ ] `Bongbetic/whatsmeow` fork exposes place/accept with opaque
      media params; `go.mod` gains the `replace`; weekly
      upstream-merge starts. **Pending — needs live verification,
      not just code.**
- [ ] Until then `CallPlace`/`CallAccept` return
      `ErrCallsForkRequired`, surfaced in the UI banner (never a
      silent no-op).

## WL-5.2 audio — Gate 5a

1:1 voice phone↔Linux 60 s, both directions audible, hangup clean.

## WL-5.3 call UI

- [x] Incoming dialog (accept/reject), call window with controls,
      call log writes on every ended leg (missed/rejected/answered).
- [ ] Reject path verified live (upstream `RejectCall`).

## WL-5.4 video — Gate 5b

1:1 video both directions.

## WL-5.5 group calls + links + waiting room / WL-5.6 screen share

- Group offer notices tracked (`CallOfferNotice`); links report
  the fork requirement honestly. Gate 5c with 5.5.

## WL-5.7 devices/echo/low-data

Stretch; H2 ML effects explicitly out.
