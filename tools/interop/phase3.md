# Phase 3 interop notes — Gate 3 (WL-3.1..WL-3.6)

## WL-3.1 voice notes

- [ ] Record 60 s voice note: OGG/Opus lands in `media/`, waveform
      stored, plays back on phone with correct duration.
- [ ] Inbound voice note: VoicePlayer plays, pause/seek work.
- [ ] Speed 1.5×/2×: DEFERRED — GtkMediaStream has no rate API;
      waits for the GStreamer-level player shared with Phase 5.

## WL-3.2 viewer + video notes + albums

- [ ] Image opens fullscreen with caption; album prev/next walks.
- [ ] Video plays with native controls; "Open with…" works.

## WL-3.3 downloads

- [ ] 20 MB video on mobile data: manual download, queue shows
      progress, manager caps at 3 concurrent (unit-proven).
- [ ] Auto-download rules match settings toggles.
- [ ] Storage page shows per-chat usage; clear-cache empties thumbs.

## WL-3.4 stickers/GIF/emoji

- [ ] Emoji picker inserts with skin tone; search filters.
- [ ] Image → sticker converts to 512 WebP and sends (ffmpeg).
- [ ] GIF sends as video with GifPlayback: DEFERRED to WL-3.4b
      (needs video-message attribute plumbing in core).

## WL-3.5 image editor v1

- [ ] Crop + rotate + draw land on the send path.
- [ ] Text annotation: DEFERRED to WL-3.5b (`ErrTextUnsupported`
      hook in place; needs a font rasterizer, no new dep yet).

## WL-3.6 sounds

- [ ] Notification plays notice.wav; sent/receive ticks audible.
- [ ] Tones are gensounds output only (CREDITS); no Meta audio.

## Portability notes (verified this phase)

- Transcode uses openh264enc+faac (both in gst-plugins-bad1, the
  pinned dep set); x264enc (ugly) and voaacenc are absent here.
- `make test` (not bare `go test`) carries `-tags sqlite_fts5`;
  CI matches. README documents this.
