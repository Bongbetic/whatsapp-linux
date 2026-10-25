package core

import (
	"context"
	"errors"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/voip"
)

// ErrCallsForkRequired is returned by call operations that need the
// Bongbetic/whatsmeow fork (WL-5.1: PlaceCall/AcceptCall plus media
// keys). Reject of an unanswered call works upstream today.
var ErrCallsForkRequired = errors.New("calls need the whatsmeow fork (WL-5.1); see tools/interop/phase5.md")

// CallPlace starts an outbound call (WL-5.1 fork).
func (e *Engine) CallPlace(ctx context.Context, chat bridge.JID, video bool) (string, error) {
	return "", ErrCallsForkRequired
}

// CallAccept accepts an incoming call with media (WL-5.1 fork).
func (e *Engine) CallAccept(ctx context.Context, callID string, video bool) error {
	return ErrCallsForkRequired
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

// CallHangup ends the local leg: unanswered legs reject upstream;
// answered media teardown needs the fork.
func (e *Engine) CallHangup(ctx context.Context, callID string) error {
	c, ok := e.calls.Get(callID)
	if !ok {
		return ErrCallsForkRequired
	}
	switch c.Stage {
	case voip.StageRinging, voip.StageOffering, voip.StagePreAccept:
		cli, err := e.client()
		if err != nil {
			return err
		}
		chat := bridge.JID{User: c.Chat}
		if err := cli.RejectCall(ctx, toWAJID(chat), callID); err != nil {
			return err
		}
		return e.calls.End(callID, "rejected")
	default:
		return ErrCallsForkRequired
	}
}

// callEventToDTO maps voip stages to UI events.
func callEventToDTO(c voip.Call) bridge.CallEvent {
	return bridge.CallEvent{ID: c.ID, Direction: c.Direction, Media: c.Media, Stage: c.Stage}
}

// onCallChange posts switch transitions to the UI and writes the
// call-log row when a call ends (WL-5.3).
func (e *Engine) onCallChange(c voip.Call) {
	e.post(callEventToDTO(c))
	if c.Stage == voip.StageEnded && e.db != nil {
		dur := int64(0)
		if !c.EndedAt.IsZero() && !c.StartedAt.IsZero() {
			dur = int64(c.EndedAt.Sub(c.StartedAt).Seconds())
		}
		_ = e.db.LogCall(context.Background(), c.ID, c.Chat, c.Direction, c.Media,
			c.StartedAt.Unix(), dur, c.Outcome)
	}
}
