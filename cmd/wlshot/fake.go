package main

import (
	"context"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// fakeEngine implements just what the GUI touches at startup and
// while viewing; any other method panics via the nil embedded
// interface, which flags a missing override.
type fakeEngine struct {
	bridge.Engine
	ev        chan bridge.Event
	loggedOut bool
}

func (f *fakeEngine) Start(ctx context.Context) error {
	st := bridge.ConnConnected
	if f.loggedOut {
		st = bridge.ConnLoggedOut
	}
	f.ev <- bridge.ConnEvent{State: st}
	return nil
}

func (f *fakeEngine) Stop() error                 { return nil }
func (f *fakeEngine) Events() <-chan bridge.Event { return f.ev }

func (f *fakeEngine) PairQR(ctx context.Context) (<-chan bridge.QRCode, error) {
	ch := make(chan bridge.QRCode, 1)
	ch <- bridge.QRCode{Code: "2@demo-not-a-real-code,abc,def,ghi"}
	return ch, nil
}

func (f *fakeEngine) MarkRead(ctx context.Context, chat bridge.JID, ids ...string) error {
	return nil
}
func (f *fakeEngine) SetTyping(ctx context.Context, chat bridge.JID, typing bool) error {
	return nil
}

func (f *fakeEngine) PrivacyGet(ctx context.Context) (bridge.PrivacySettings, error) {
	return bridge.PrivacySettings{"lastseen": "contacts", "profile": "contacts", "readreceipts": "all", "groupadd": "contacts", "online": "all"}, nil
}

func (f *fakeEngine) BlocklistGet(ctx context.Context) ([]bridge.JID, error) { return nil, nil }

func (f *fakeEngine) StatusPost(ctx context.Context, text string) error { return nil }
