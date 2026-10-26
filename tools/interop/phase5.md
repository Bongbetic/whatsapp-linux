# Phase 5 interop notes — Gates 5a/5b/5c (WL-5.0..WL-5.7)

## WL-5.0 signaling probe (media plan A/B decision)

- [ ] `WLPROBE_HOME=$(mktemp -d) go run ./cmd/wlprobe call-probe`
      while the phone rings this device: `calls.log` shows
      offer → (preaccept) → terminate with IDs and relay hosts.
- [ ] Decision: plan A (raw RTP/SRTP, `voip` pipeline consts) vs
      plan B (webrtcbin) from relay/ICE evidence. **Pending live
      capture — default stays plan A.**

## WL-5.1 fork: PlaceCall/AcceptCall

- [x] `Bongbetic/whatsmeow` fork exposes place/preaccept/accept/
      terminate (`call_place.go`, framing mirrored from upstream
      `RejectCall` + inbound shapes); `go.mod` gains the `replace`;
      weekly upstream-merge starts at the next pin bump.
      **Needs live verification, not just code.**
- [x] Engine `CallPlace`/`CallAccept`/`CallHangup` wired to the
      fork (no more `ErrCallsForkRequired`); transport stanza →
      relay parse (`voip.FromNode`) → GStreamer audio leg;
      ambiguous shapes stay signaling-only with the stanza logged.
- [x] Call window: real stages, mute leg toggle, hangup; incoming
      ringtone loop (`ringtone.wav` via paplay); call log writes
      on every ended leg.
- [ ] Owner live test (Gate 5a): place Linux→phone — does the
      phone ring? Accept on phone — do accept+transport events
      arrive (`--debug` stanza log)? 60 s both directions audible?
      Then reverse (phone→Linux accept). Paste `--debug` `call
      stanza` lines here when the framing needs iteration.

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
