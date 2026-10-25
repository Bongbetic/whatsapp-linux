// Package core is the whatsmeow wrapper (plan §7): connection,
// event→DTO conversion, and send paths. It is the only package
// that imports whatsmeow. UI code must use bridge.Engine.
package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
	"github.com/Bongbetic/whatsapp-linux/internal/voip"
)

// Compile-time check: Engine implements bridge.Engine.
var _ bridge.Engine = (*Engine)(nil)

// Engine wraps one whatsmeow Client plus app.db writes (§8).
type Engine struct {
	log zerolog.Logger
	db  *appstore.DB

	mu        sync.Mutex
	cli       *whatsmeow.Client
	container *sqlstore.Container
	events    chan bridge.Event
	mediaMu   sync.Mutex
	media     map[string]whatsmeow.DownloadableMessage // chatID/msgID → proto
	pool      [2]chan dispatchItem
	poolWg    sync.WaitGroup
	calls     *voip.Switch
}

type dispatchItem struct {
	chat string
	fn   func()
}

// New builds an Engine. Profile dir holds session.db + app.db (§6).
// The whatsmeow container is opened lazily in Start so unit tests
// can exercise conversion without a database.
func New(log zerolog.Logger, profileDir string, db *appstore.DB) (*Engine, error) {
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return nil, err
	}
	e := &Engine{
		log:    log,
		db:     db,
		events: make(chan bridge.Event, 256),
		media:  make(map[string]whatsmeow.DownloadableMessage),
	}
	e.calls = voip.NewSwitch(e.onCallChange)
	// §8: worker pool of 2, ordered per chat by JID hash.
	for i := 0; i < 2; i++ {
		e.pool[i] = make(chan dispatchItem, 128)
		e.poolWg.Add(1)
		go e.worker(e.pool[i])
	}
	return e, nil
}

func (e *Engine) worker(ch chan dispatchItem) {
	defer e.poolWg.Done()
	for item := range ch {
		start := time.Now()
		item.fn()
		if d := time.Since(start); d > 2*time.Second {
			e.log.Warn().Dur("took", d).Msg("slow handler past 2s budget (§8)")
		}
	}
}

// openContainer opens session.db (D-4: owned by whatsmeow sqlstore).
func (e *Engine) openContainer(ctx context.Context, profileDir string) error {
	addr := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL",
		filepath.Join(profileDir, "session.db"))
	container, err := sqlstore.New(ctx, "sqlite3", addr, waAdapter{log: e.log})
	if err != nil {
		return err
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return err
	}
	// D-10: desktop companion identity (whatsmeow defaults).
	store.SetOSInfo(bridge.DevicePlatform, [3]uint32{1, 0, 0})
	e.mu.Lock()
	e.container = container
	e.cli = whatsmeow.NewClient(device, waAdapter{log: e.log})
	e.cli.AddEventHandler(e.handle)
	e.mu.Unlock()
	return nil
}

// Start connects (or leaves disconnected for pairing if logged out).
func (e *Engine) Start(ctx context.Context) error {
	e.post(bridge.ConnEvent{State: bridge.ConnConnecting})
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return fmt.Errorf("engine not opened: call Open first")
	}
	if cli.Store.ID == nil {
		// Fresh device: stay offline until PairQR/PairPhone.
		e.post(bridge.ConnEvent{State: bridge.ConnLoggedOut})
		return nil
	}
	if err := cli.Connect(); err != nil {
		e.post(bridge.ConnEvent{State: bridge.ConnOffline})
		return err
	}
	return nil
}

// Open wires the Engine to a profile dir (session.db). Split from
// Start so wlprobe and tests control each step.
func (e *Engine) Open(ctx context.Context, profileDir string) error {
	return e.openContainer(ctx, profileDir)
}

// Stop shuts down: sends stop → disconnect (≤5 s) (§8).
func (e *Engine) Stop() error {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli != nil && cli.IsConnected() {
		cli.Disconnect()
	}
	return nil
}

// Events returns the DTO event channel (§8: buffered 256).
func (e *Engine) Events() <-chan bridge.Event { return e.events }

// post emits, dropping only presence/typing at 90% buffer (§8).
func (e *Engine) post(ev bridge.Event) {
	droppable := ev.Kind() == "typing"
	select {
	case e.events <- ev:
	default:
		if droppable || len(e.events)*10 >= cap(e.events)*9 {
			return
		}
		e.events <- ev // messages never drop: block
	}
}

// Logout logs out and clears the session.
func (e *Engine) Logout(ctx context.Context) error {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return nil
	}
	return cli.Logout(ctx)
}

// PairQR streams pairing codes for a fresh device. whatsmeow requires
// GetQRChannel BEFORE Connect; Connect then runs in the background
// while codes (then success/timeout) arrive on the channel.
func (e *Engine) PairQR(ctx context.Context) (<-chan bridge.QRCode, error) {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return nil, fmt.Errorf("engine not opened")
	}
	if cli.Store.ID != nil {
		return nil, fmt.Errorf("already paired: logout first to re-pair")
	}
	ch, err := cli.GetQRChannel(ctx)
	if err != nil {
		return nil, err
	}
	go func() {
		if err := cli.Connect(); err != nil {
			e.log.Error().Err(err).Msg("pair connect")
		}
	}()
	out := make(chan bridge.QRCode, 4)
	go func() {
		defer close(out)
		for item := range ch {
			switch item.Event {
			case "code":
				out <- bridge.QRCode{Code: item.Code, Timeout: item.Timeout}
			case "success":
				e.post(bridge.PairDone{OK: true})
				return
			case "timeout", "err-client-outdated":
				if item.Error != nil {
					e.log.Error().Err(item.Error).Str("event", item.Event).Msg("pairing failed")
				}
				e.post(bridge.PairDone{OK: false})
				return
			}
		}
	}()
	return out, nil
}

// PairPhone returns the 8-char link code for phone-number pairing.
func (e *Engine) PairPhone(ctx context.Context, number string) (string, error) {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return "", fmt.Errorf("engine not opened")
	}
	if !cli.IsConnected() {
		if err := cli.Connect(); err != nil {
			return "", err
		}
	}
	return cli.PairPhone(ctx, number, true, whatsmeow.PairClientChrome, "whatsapp-linux")
}

// SendText sends a text message, optimistically (§8: caller writes
// the 'sending' row; we confirm with the stanza id).
func (e *Engine) SendText(ctx context.Context, chat bridge.JID, body string, opts bridge.SendOpts) (bridge.MsgID, error) {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return "", fmt.Errorf("engine not opened")
	}
	resp, err := cli.SendMessage(ctx, toWAJID(chat), textMessage(body, opts))
	if err != nil {
		return "", err
	}
	return bridge.MsgID(resp.ID), nil
}

// SendMedia uploads and sends one attachment (WL-1.4).
func (e *Engine) SendMedia(ctx context.Context, chat bridge.JID, f bridge.MediaFile, opts bridge.SendOpts) (bridge.MsgID, error) {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return "", fmt.Errorf("engine not opened")
	}
	raw, err := os.ReadFile(f.Path)
	if err != nil {
		return "", err
	}
	built, err := buildMediaMessage(ctx, cli, f, raw, opts)
	if err != nil {
		return "", err
	}
	resp, err := cli.SendMessage(ctx, toWAJID(chat), built)
	if err != nil {
		return "", err
	}
	return bridge.MsgID(resp.ID), nil
}

// DownloadMedia fetches a cached inbound attachment (§8: max 3
// concurrent enforced by caller queue; cache holds recent protos).
func (e *Engine) DownloadMedia(ctx context.Context, ref bridge.MediaRef, w io.Writer) error {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return fmt.Errorf("engine not opened")
	}
	e.mediaMu.Lock()
	dm, ok := e.media[ref.Chat.String()+"/"+ref.MsgID]
	e.mediaMu.Unlock()
	if !ok {
		return fmt.Errorf("media expired from cache; re-request from sender (WL-3.3)")
	}
	raw, err := cli.Download(ctx, dm)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

// MarkRead sends read receipts.
func (e *Engine) MarkRead(ctx context.Context, chat bridge.JID, ids ...string) error {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return fmt.Errorf("engine not opened")
	}
	wids := make([]types.MessageID, 0, len(ids))
	for _, id := range ids {
		wids = append(wids, types.MessageID(id))
	}
	return cli.MarkRead(ctx, wids, time.Now(), toWAJID(chat), types.JID{})
}

// SetTyping sends composing/paused chat presence (3 s debounce by caller).
func (e *Engine) SetTyping(ctx context.Context, chat bridge.JID, typing bool) error {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return fmt.Errorf("engine not opened")
	}
	state := typeComposing(!typing)
	return cli.SendChatPresence(ctx, toWAJID(chat), state, typeMediaText())
}

// handle is the single whatsmeow event entry: convert → store → post (§8).
func (e *Engine) handle(raw any) {
	switch evt := raw.(type) {
	case *events.Connected:
		e.post(bridge.ConnEvent{State: bridge.ConnConnected})
	case *events.Disconnected:
		e.post(bridge.ConnEvent{State: bridge.ConnOffline})
	case *events.LoggedOut:
		e.post(bridge.ConnEvent{State: bridge.ConnLoggedOut})
	case *events.Message:
		item := dispatchItem{chat: evt.Info.Chat.String(), fn: func() {
			m := messageToDTO(evt)
			if e.db != nil {
				e.ensureChat(context.Background(), m.Chat, evt)
				if err := e.db.UpsertMessage(context.Background(), m); err != nil {
					e.log.Error().Err(err).Msg("upsert inbound message")
				}
				if dm, ok := firstDownloadable(evt.Message); ok {
					e.cacheMedia(m, dm)
				}
			}
			e.post(bridge.MessageEvent{Msg: m})
		}}
		select {
		case e.pool[item.poolIndex()] <- item:
		default:
			e.pool[item.poolIndex()] <- item // messages never drop
		}
	case *events.Receipt:
		e.post(receiptToDTO(evt))
	case *events.HistorySync:
		e.ingestHistorySync(evt)
	case *events.GroupInfo:
		e.post(groupInfoToDTO(evt))
	case *events.ChatPresence:
		e.post(bridge.TypingEvent{Chat: fromWAJID(evt.MessageSource.Chat), Sender: fromWAJID(evt.MessageSource.Sender), Typing: evt.State == chatPresenceComposing()})
	case *events.CallOffer:
		_, _ = e.calls.Ring(evt.CallID, evt.From.String(), "audio")
	case *events.CallOfferNotice:
		_, _ = e.calls.Ring(evt.CallID, evt.From.String(), evt.Media)
	case *events.CallPreAccept:
		_ = e.calls.PreAccept(evt.CallID)
	case *events.CallAccept:
		_ = e.calls.PreAccept(evt.CallID)
		_ = e.calls.Connected(evt.CallID)
	case *events.CallTransport:
		_ = e.calls.Transport(evt.CallID)
	case *events.CallTerminate:
		outcome := "answered"
		if c, ok := e.calls.Get(evt.CallID); ok && c.Stage == voip.StageRinging {
			outcome = "missed"
		}
		_ = e.calls.End(evt.CallID, outcome)
	case *events.CallReject:
		_ = e.calls.End(evt.CallID, "rejected")
	}
}

// ensureChat creates the chat row for unknown chats on first message.
func (e *Engine) ensureChat(ctx context.Context, chat bridge.JID, evt *events.Message) {
	if e.db == nil {
		return
	}
	if _, err := e.db.GetChat(ctx, chat.String()); err == nil {
		return
	}
	kind := bridge.ChatDM
	if chat.Server == "g.us" {
		kind = bridge.ChatGroup
	} else if chat.Server == "newsletter" {
		kind = bridge.ChatChannel
	} else if chat.Server == "status" {
		kind = bridge.ChatStatusBroadcast
	}
	_ = e.db.UpsertChat(ctx, bridge.Chat{JID: chat, Kind: kind, Title: evt.Info.PushName})
}

// cacheMedia keeps the downloadable proto for recent inbound media
// so DownloadMedia can fetch bytes without a re-request (WL-1.4).
func (e *Engine) cacheMedia(m bridge.Message, dm whatsmeow.DownloadableMessage) {
	e.mediaMu.Lock()
	defer e.mediaMu.Unlock()
	if len(e.media) > 512 {
		e.media = make(map[string]whatsmeow.DownloadableMessage, 512)
	}
	e.media[m.Chat.String()+"/"+m.ID] = dm
}

// poolIndex hashes the chat to a worker (ordered per chat, §8).
func (d dispatchItem) poolIndex() int {
	h := 0
	for i := 0; i < len(d.chat); i++ {
		h = h*31 + int(d.chat[i])
	}
	if h < 0 {
		h = -h
	}
	return h % 2
}

// waAdapter bridges zerolog to waLog.Logger with D-9 scrubbing.
type waAdapter struct{ log zerolog.Logger }

func (a waAdapter) emit(e *zerolog.Event, msg string, args ...any) {
	e.Msg(logx.Scrub(fmt.Sprintf(msg, args...)))
}

func (a waAdapter) Warnf(msg string, args ...any)  { a.emit(a.log.Warn(), msg, args...) }
func (a waAdapter) Errorf(msg string, args ...any) { a.emit(a.log.Error(), msg, args...) }
func (a waAdapter) Infof(msg string, args ...any)  { a.emit(a.log.Info(), msg, args...) }
func (a waAdapter) Debugf(msg string, args ...any) { a.emit(a.log.Debug(), msg, args...) }
func (a waAdapter) Sub(module string) waLog.Logger {
	return waAdapter{log: a.log.With().Str("mod", module).Logger()}
}
