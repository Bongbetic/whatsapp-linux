// Package voip implements the call state machine and media
// parameters (plan §13). Signaling place/accept needs the
// Bongbetic/whatsmeow fork (WL-5.1); this package drives everything
// upstream events already provide (offer/ring/terminate) and owns
// the state transitions the fork will feed.
package voip

import (
	"fmt"
	"sync"
	"time"
)

// Stages per §13: idle → offering|ringing → preaccept →
// connecting(relay) → in-call → terminating → ended.
const (
	StageIdle        = "idle"
	StageOffering    = "offering"
	StageRinging     = "ringing"
	StagePreAccept   = "preaccept"
	StageConnecting  = "connecting"
	StageInCall      = "in-call"
	StageTerminating = "terminating"
	StageEnded       = "ended"
)

// Media pipelines: plan A = raw RTP/SRTP elements (keys from
// signaling, relay endpoints from CallTransport). %s slots are
// (srtp key, relay host, relay port). Plan B (webrtcbin) is taken
// only if WL-5.0 interop shows relays expect DTLS-SRTP + ICE.
// Echo: PipeWire echo-cancel module when present, else webrtcdsp
// (gst-plugins-bad). Screen share (WL-5.6): portal ScreenCast →
// PipeWire node → second video track.
const (
	AudioSendPipeline = `autoaudiosrc ! audioconvert ! audioresample ! ` +
		`opusenc ! rtpopuspay ! srtpenc key="%s" ! udpsink host=%s port=%d`
	AudioRecvPipeline = `udpsrc port=%d ! srtpdec key="%s" ! ` +
		`rtpopusdepay ! opusdec ! audioconvert ! autoaudiosink`
	VideoSendPipeline = `autoaudiosrc name=vs ! videoconvert ! %s ! ` +
		`rtpvp8pay ! srtpenc key="%s" ! udpsink host=%s port=%d`
)

// Call is one call's state.
type Call struct {
	ID        string
	Chat      string
	Direction string // in|out
	Media     string // audio|video
	Stage     string
	StartedAt time.Time
	EndedAt   time.Time
	Outcome   string // answered|missed|rejected|failed (log row at ended)
}

// Switch multiplexes calls by ID with serialized transitions.
type Switch struct {
	mu       sync.Mutex
	calls    map[string]*Call
	OnChange func(Call)
}

// NewSwitch builds an empty switch.
func NewSwitch(onChange func(Call)) *Switch {
	return &Switch{calls: map[string]*Call{}, OnChange: onChange}
}

func (s *Switch) emit(c *Call) {
	if s.OnChange != nil {
		s.OnChange(*c)
	}
}

// Offer starts an outgoing call (idle → offering).
func (s *Switch) Offer(id, chat, media string) (*Call, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.calls[id]; ok {
		return nil, fmt.Errorf("call %s already tracked", id)
	}
	c := &Call{ID: id, Chat: chat, Direction: "out", Media: media, Stage: StageOffering, StartedAt: time.Now()}
	s.calls[id] = c
	s.emit(c)
	return c, nil
}

// Ring registers an incoming call (idle → ringing).
func (s *Switch) Ring(id, chat, media string) (*Call, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.calls[id]; ok {
		return nil, fmt.Errorf("call %s already tracked", id)
	}
	c := &Call{ID: id, Chat: chat, Direction: "in", Media: media, Stage: StageRinging, StartedAt: time.Now()}
	s.calls[id] = c
	s.emit(c)
	return c, nil
}

// PreAccept moves offering|ringing → preaccept.
func (s *Switch) PreAccept(id string) error {
	return s.move(id, StagePreAccept, []string{StageOffering, StageRinging})
}

// Transport moves preaccept → connecting (relay endpoints known).
func (s *Switch) Transport(id string) error {
	return s.move(id, StageConnecting, []string{StagePreAccept, StageConnecting})
}

// Connected moves connecting → in-call (media flowing).
func (s *Switch) Connected(id string) error {
	return s.move(id, StageInCall, []string{StageConnecting, StagePreAccept})
}

// Hangup moves any live stage → terminating (local hangup).
func (s *Switch) Hangup(id string) error {
	return s.move(id, StageTerminating, []string{StageOffering, StageRinging, StagePreAccept, StageConnecting, StageInCall})
}

// End moves → ended with an outcome (remote terminate / cleanup).
func (s *Switch) End(id, outcome string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[id]
	if !ok {
		return fmt.Errorf("unknown call %s", id)
	}
	if c.Stage == StageEnded {
		return nil
	}
	c.Stage, c.Outcome, c.EndedAt = StageEnded, outcome, time.Now()
	s.emit(c)
	return nil
}

// Get returns a copy of one call.
func (s *Switch) Get(id string) (Call, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[id]
	if !ok {
		return Call{}, false
	}
	return *c, true
}

func (s *Switch) move(id, to string, from []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.calls[id]
	if !ok {
		return fmt.Errorf("unknown call %s", id)
	}
	for _, f := range from {
		if c.Stage == f {
			c.Stage = to
			s.emit(c)
			return nil
		}
	}
	return fmt.Errorf("call %s: %s → %s illegal", id, c.Stage, to)
}
