package core

import (
	"context"
	"errors"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	waBinary "go.mau.fi/whatsmeow/binary"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
	"github.com/Bongbetic/whatsapp-linux/internal/voip"
)

// CallPlace starts an outbound call via the Bongbetic/whatsmeow
// fork (WL-5.1): sends the offer, tracks the leg in the switch.
// The peer ringing (then preaccept → transport events) is the live
// proof; see tools/interop/phase5.md.
func (e *Engine) CallPlace(ctx context.Context, chat bridge.JID, video bool) (string, error) {
	cli, err := e.client()
	if err != nil {
		return "", err
	}
	id, err := cli.PlaceCall(ctx, toWAJID(chat), video)
	if err != nil {
		return "", err
	}
	media := map[bool]string{true: "video", false: "audio"}[video]
	if _, oerr := e.calls.Offer(id, chat.String(), media); oerr != nil {
		e.log.Error().Err(oerr).Msg("call switch offer")
	}
	return id, nil
}

// CallAccept accepts an incoming call with media (WL-5.1 fork):
// preaccept first (ringing → preaccept), then accept. Transport and
// media flow follow on CallTransport events.
func (e *Engine) CallAccept(ctx context.Context, callID string, video bool) error {
	c, ok := e.calls.Get(callID)
	if !ok {
		return errors.New("unknown call " + callID)
	}
	cli, err := e.client()
	if err != nil {
		return err
	}
	peer := toWAJID(parseWAJID(c.Chat))
	if perr := cli.PreAcceptCall(ctx, peer, callID); perr != nil {
		return perr
	}
	_ = e.calls.PreAccept(callID)
	if err := cli.AcceptCall(ctx, peer, callID, video); err != nil {
		return err
	}
	return nil
}

// SetMuted mutes (hours>0, 0 = forever) or unmutes a chat via
// app-state sync; the Mute echo event updates the local row (WL-2.6).
func (e *Engine) SetMuted(ctx context.Context, chat bridge.JID, muted bool, hours int64) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	var dur time.Duration
	if muted && hours > 0 {
		dur = time.Duration(hours) * time.Hour
	}
	if err := cli.SendAppState(ctx, appstate.BuildMute(toWAJID(chat), muted, dur)); err != nil {
		return err
	}
	// Optimistic local mirror; the echo event confirms.
	if e.db != nil {
		until := int64(-1)
		if !muted {
			until = 0
		} else if hours > 0 {
			until = time.Now().Add(dur).Unix()
		}
		if uerr := e.db.SetChatMuted(ctx, chat.String(), until); uerr == nil {
			e.post(bridge.RefreshEvent{})
		}
	}
	return nil
}

// CallReject rejects an incoming (or cancels an outgoing unanswered)
// call — works upstream via RejectCall.
func (e *Engine) CallReject(ctx context.Context, chat bridge.JID, callID string) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	if err := cli.RejectCall(ctx, toWAJID(chat), callID); err != nil {
		return err
	}
	_ = e.calls.End(callID, "rejected")
	return nil
}

// CallHangup ends the local leg: unanswered legs reject upstream
// (proven path); live legs terminate via the fork (WL-5.1).
func (e *Engine) CallHangup(ctx context.Context, callID string) error {
	c, ok := e.calls.Get(callID)
	if !ok {
		return errors.New("unknown call " + callID)
	}
	cli, err := e.client()
	if err != nil {
		return err
	}
	peer := toWAJID(parseWAJID(c.Chat))
	switch c.Stage {
	case voip.StageRinging, voip.StageOffering, voip.StagePreAccept:
		if err := cli.RejectCall(ctx, peer, callID); err != nil {
			return err
		}
		return e.calls.End(callID, "rejected")
	default:
		if err := cli.TerminateCall(ctx, peer, callID, "user-hangup"); err != nil {
			return err
		}
		return e.calls.End(callID, "answered")
	}
}

// callEventToDTO maps voip stages to UI events.
func callEventToDTO(c voip.Call) bridge.CallEvent {
	return bridge.CallEvent{ID: c.ID, Direction: c.Direction, Media: c.Media, Stage: c.Stage}
}

// logCallNode records the raw signaling stanza for WL-5.0 evidence
// (scrubbed; run with --debug while the phone rings this device, then
// attach the lines to tools/interop/phase5.md).
func (e *Engine) logCallNode(dir string, n interface{ String() string }) {
	if n == nil {
		return
	}
	e.log.Debug().Str("dir", dir).Str("node", logx.Scrub(n.String())).Msg("call stanza")
}

// onCallChange posts switch transitions to the UI and writes the
// call-log row when a call ends (WL-5.3).
// startMedia brings up call audio from a transport stanza (WL-5.2):
// the first unambiguous relay triple wins; anything else stays
// signaling-only and the stanza log owns the evidence.
func (e *Engine) startMedia(callID string, data *waBinary.Node) {
	relays := voip.FromNode(data)
	if len(relays) == 0 {
		e.log.Debug().Str("call", callID).Msg("no relay triple in transport; media stays down")
		return
	}
	r := relays[0]
	e.log.Info().Str("call", callID).Str("host", logx.Scrub(r.Host)).Int("port", r.Port).Msg("starting call audio")
	leg := voip.DialAudio(r.Host, r.Port, r.Key)
	if err := leg.Start(); err != nil {
		e.log.Error().Err(err).Msg("call audio start")
		return
	}
	e.legsMu.Lock()
	if old, ok := e.legs[callID]; ok {
		old.Stop()
	}
	e.legs[callID] = leg
	e.legsMu.Unlock()
}

// stopMedia ends call audio (idempotent).
func (e *Engine) stopMedia(callID string) {
	e.legsMu.Lock()
	leg, ok := e.legs[callID]
	if ok {
		delete(e.legs, callID)
	}
	e.legsMu.Unlock()
	if ok {
		leg.Stop()
	}
}

// SetCallMuted toggles the microphone leg of a live call.
func (e *Engine) SetCallMuted(callID string, muted bool) {
	e.legsMu.Lock()
	leg, ok := e.legs[callID]
	e.legsMu.Unlock()
	if ok {
		leg.SetMuted(muted)
	}
}

func (e *Engine) onCallChange(c voip.Call) {
	e.post(callEventToDTO(c))
	if c.Stage == voip.StageEnded {
		e.stopMedia(c.ID)
	}
	if c.Stage == voip.StageEnded && e.db != nil {
		dur := int64(0)
		if !c.EndedAt.IsZero() && !c.StartedAt.IsZero() {
			dur = int64(c.EndedAt.Sub(c.StartedAt).Seconds())
		}
		_ = e.db.LogCall(context.Background(), c.ID, c.Chat, c.Direction, c.Media,
			c.StartedAt.Unix(), dur, c.Outcome)
	}
}
