// Package app owns the Gtk.Application lifecycle (WL-2.1): single
// instance, main window (1180×760, header bar with connection dot,
// two-pane Paned), login↔main view swap, event dispatcher (§8: core
// channel → glib.IdleAdd), global shortcuts, close-to-tray.
package app

import (
	"context"
	"strconv"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/rs/zerolog"

	"github.com/Bongbetic/whatsapp-linux/internal/autostart"
	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
	"github.com/Bongbetic/whatsapp-linux/internal/notify"
	"github.com/Bongbetic/whatsapp-linux/internal/settings"
	"github.com/Bongbetic/whatsapp-linux/internal/store"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/chatinfo"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/chatlist"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/convo"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/dialogs"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/login"
	uisettings "github.com/Bongbetic/whatsapp-linux/internal/ui/settings"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// App is the running GUI shell.
type App struct {
	gtkApp   *gtk.Application
	engine   bridge.Engine
	db       *store.DB
	settings *settings.Store
	log      zerolog.Logger
	version  string
	hidden   bool

	win     *gtk.ApplicationWindow
	connLbl *gtk.Label
	banner  *gtk.Revealer
	bannerL *gtk.Label
	stack   *gtk.Stack
	paned   *gtk.Paned
	loginV  *login.View
	chatsV  *chatlist.View
	convV   *convo.View
	infoV   *chatinfo.View
	notif   *notify.Notifier

	current bridge.Chat
	hasChat bool
}

// Config wires the shell to the backend.
type Config struct {
	Engine   bridge.Engine
	DB       *store.DB
	Settings *settings.Store
	Log      zerolog.Logger
	Version  string
	Hidden   bool
}

// Run starts the application (blocks until quit). No GTK calls happen
// before this, so headless commands can build Config without a display.
func Run(cfg Config) int {
	a := &App{engine: cfg.Engine, db: cfg.DB, settings: cfg.Settings, log: cfg.Log, version: cfg.Version, hidden: cfg.Hidden}
	a.gtkApp = gtk.NewApplication(bridge.AppID, gio.ApplicationHandlesOpen)
	a.gtkApp.ConnectActivate(a.activate)
	return a.gtkApp.Run(nil)
}

func (a *App) activate() {
	a.win = gtk.NewApplicationWindow(a.gtkApp)
	a.win.SetTitle(bridge.DisplayName)
	a.win.SetDefaultSize(a.settings.GetInt(settings.KeyWindowW), a.settings.GetInt(settings.KeyWindowH))

	bar := gtk.NewHeaderBar()
	a.connLbl = gtk.NewLabel("●")
	a.connLbl.SetTooltipText("Connecting…")
	bar.PackStart(a.connLbl)
	a.win.SetTitlebar(bar)

	a.banner, a.bannerL = widgets.Banner()
	root := gtk.NewBox(gtk.OrientationVertical, 0)
	root.Append(a.banner)

	a.stack = gtk.NewStack()
	main := gtk.NewBox(gtk.OrientationVertical, 0)
	a.paned = gtk.NewPaned(gtk.OrientationHorizontal)

	a.chatsV = chatlist.New(a.openChat)
	a.paned.SetStartChild(a.chatsV.Box)

	right := gtk.NewBox(gtk.OrientationHorizontal, 0)
	a.convV = convo.New(convo.Callbacks{
		OnSend:   a.send,
		OnTyping: func(chat bridge.JID, typing bool) { _ = a.engine.SetTyping(context.Background(), chat, typing) },
		OnDraft: func(chat bridge.JID, text string) {
			_ = a.db.UpsertChat(context.Background(), bridge.Chat{JID: chat, Draft: text})
		},
		OnAttach: func(chat bridge.JID) { a.log.Info().Msg("attach picker lands at WL-2.3") },
		OnCall: func(chat bridge.JID, video bool) {
			kind := "audio"
			if video {
				kind = "video"
			}
			a.log.Info().Str("kind", kind).Msg("calls land at WL-5.x")
		},
	}, a.settings.GetBool(settings.KeyEnterToSend))
	right.Append(a.convV.Box)
	a.convV.Box.SetHExpand(true)
	a.infoV = chatinfo.New(chatinfo.Callbacks{
		OnMute:      func(chat bridge.JID, h int64) { a.log.Info().Msg("mute lands at WL-2.6") },
		OnBlock:     func(chat bridge.JID, b bool) { a.log.Info().Msg("block lands at WL-4.6") },
		OnGroupEdit: func(chat bridge.JID) { a.log.Info().Msg("group mgmt lands at WL-4.1") },
	})
	right.Append(a.infoV.Box)
	a.paned.SetEndChild(right)
	main.Append(a.paned)
	a.stack.AddNamed(main, "main")

	a.loginV = login.New(
		func(number string) string {
			code, err := a.engine.PairPhone(context.Background(), number)
			if err != nil {
				a.loginV.SetStatus("Link code failed: " + err.Error())
				return ""
			}
			return code
		},
		func() { _ = a.engine.Stop() },
	)
	a.stack.AddNamed(a.loginV.Box, "login")

	a.stack.AddNamed(uisettings.New(a.settings, func() {
		a.log.Info().Msg("privacy pages land at WL-4.6")
	}).Box, "settings")

	root.Append(a.stack)
	a.win.SetChild(root)

	a.notif = notify.New(a.gtkApp, a.settings, a.log, notify.NoopTray{},
		func(chatJID, text string) { a.log.Info().Msg("inline reply lands at WL-2.5") },
		func(chatJID string) { _ = a.engine.MarkRead(context.Background(), parseJID(chatJID)) },
	)

	a.win.ConnectCloseRequest(func() bool {
		if a.settings.GetBool(settings.KeyCloseToTray) {
			a.win.SetVisible(false) // keep running in tray (WL-2.5)
			return true
		}
		return false
	})
	a.shortcuts()

	if !a.hidden {
		a.win.Show()
	}
	_ = autostart.Set(a.settings.GetBool(settings.KeyAutostart))
	a.showLogin("Connecting…")
	go a.dispatch()
	if err := a.engine.Start(context.Background()); err != nil {
		a.log.Error().Err(err).Msg("engine start")
	}
}

// shortcuts wires the §9 global keys.
func (a *App) shortcuts() {
	keys := gtk.NewEventControllerKey()
	keys.ConnectKeyPressed(func(keyval, _ uint, state gdk.ModifierType) bool {
		if state&gdk.ControlMask == 0 {
			return false
		}
		switch keyval {
		case gdk.KEY_n, gdk.KEY_N:
			dialogs.NewChat(a.win, func(phone string) { a.log.Info().Msg("new chat lands at WL-2.6") })
			return true
		case gdk.KEY_comma:
			a.stack.SetVisibleChildName("settings")
			return true
		}
		return false
	})
	a.win.AddController(keys)
}

// dispatch forwards backend events to GTK via glib.IdleAdd (§8).
func (a *App) dispatch() {
	for ev := range a.engine.Events() {
		ev := ev
		glib.IdleAdd(func() bool {
			a.handle(ev)
			return false
		})
	}
}

func (a *App) handle(ev bridge.Event) {
	ctx := context.Background()
	switch e := ev.(type) {
	case bridge.ConnEvent:
		a.onConn(e.State)
	case bridge.QRCode:
		a.showLogin("Scan the QR code with WhatsApp → Linked devices")
		a.loginV.ShowQR(e.Code)
	case bridge.PairDone:
		if e.OK {
			a.showMain()
		} else {
			a.showLogin("Pairing failed — restarting…")
		}
	case bridge.MessageEvent:
		a.refreshChats()
		if a.hasChat && e.Msg.Chat == a.current.JID {
			a.convV.AddMessage(e.Msg)
			_ = a.engine.MarkRead(ctx, e.Msg.Chat, e.Msg.ID)
		} else {
			a.notif.MessageArrived(chatFor(ctx, a.db, e.Msg.Chat), e.Msg)
		}
	case bridge.ChatEvent:
		a.refreshChats()
	case bridge.ReceiptEvent:
		if a.hasChat && e.Chat == a.current.JID {
			a.reloadCurrent()
		}
	case bridge.TypingEvent:
		if a.hasChat && e.Chat == a.current.JID {
			if e.Typing {
				a.convV.SetPresence("typing…")
			} else {
				a.convV.SetPresence("")
			}
		}
	case bridge.SyncProgress:
		if e.Percent < 100 {
			widgets.SetBanner(a.banner, a.bannerL, progressText(e.Percent))
		} else {
			widgets.SetBanner(a.banner, a.bannerL, "")
			a.refreshChats()
		}
	}
}

func progressText(p int) string { return "Syncing history… " + strconv.Itoa(p) + "%" }

func (a *App) onConn(s bridge.ConnState) {
	switch s {
	case bridge.ConnConnected:
		a.connLbl.SetText("●")
		a.connLbl.SetTooltipText("Connected")
		widgets.SetBanner(a.banner, a.bannerL, "")
		a.showMain()
	case bridge.ConnConnecting:
		a.connLbl.SetTooltipText("Connecting…")
		widgets.SetBanner(a.banner, a.bannerL, "Connecting…")
	case bridge.ConnOffline:
		a.connLbl.SetTooltipText("Offline — retrying")
		widgets.SetBanner(a.banner, a.bannerL, "Offline — retrying…")
	case bridge.ConnLoggedOut:
		a.showLogin("Scan the QR code with WhatsApp → Linked devices")
		go func() {
			codes, err := a.engine.PairQR(context.Background())
			if err != nil {
				a.log.Error().Err(err).Msg("pair channel")
				return
			}
			for c := range codes {
				c := c
				glib.IdleAdd(func() bool {
					a.loginV.ShowQR(c.Code)
					return false
				})
			}
		}()
	}
}

func (a *App) showLogin(status string) {
	a.loginV.SetStatus(status)
	a.stack.SetVisibleChildName("login")
}

func (a *App) showMain() {
	a.refreshChats()
	a.stack.SetVisibleChildName("main")
}

func (a *App) refreshChats() {
	chats, err := a.db.ListChats(context.Background(), 200, 0, "", false)
	if err != nil {
		a.log.Error().Err(err).Msg("list chats")
		return
	}
	a.chatsV.SetChats(chats)
}

func (a *App) openChat(c bridge.Chat) {
	a.stack.SetVisibleChildName("main")
	a.current, a.hasChat = c, true
	msgs, err := a.db.ListMessages(context.Background(), c.JID.String(), 50, 0, "")
	if err != nil {
		a.log.Error().Err(err).Msg("list messages")
	}
	// ListMessages is newest-first; reverse for display.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	a.convV.OpenChat(c, msgs)
	a.notif.Clear()
}

func (a *App) reloadCurrent() {
	if !a.hasChat {
		return
	}
	msgs, err := a.db.ListMessages(context.Background(), a.current.JID.String(), 50, 0, "")
	if err != nil {
		return
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	a.convV.OpenChat(a.current, msgs)
}

// send is the optimistic send path (§8): 'sending' row first, engine
// confirms, receipts advance state via MessageEvent/ReceiptEvent.
func (a *App) send(chat bridge.JID, text, quotedID string) {
	ctx := context.Background()
	now := time.Now()
	id := "local-" + strconv.FormatInt(now.UnixNano(), 10)
	m := bridge.Message{Chat: chat, ID: id, IsFromMe: true, Kind: bridge.MsgText, Body: text, QuotedID: quotedID, State: bridge.StateSending, Ts: now}
	if err := a.db.UpsertMessage(ctx, m); err != nil {
		a.log.Error().Err(err).Msg("optimistic insert")
		return
	}
	a.convV.AddMessage(m)
	go func() {
		sentID, err := a.engine.SendText(ctx, chat, text, bridge.SendOpts{QuotedID: quotedID})
		glib.IdleAdd(func() bool {
			if err != nil {
				a.log.Error().Err(err).Msg("send failed")
				_ = a.db.UpdateMessageState(ctx, chat.String(), id, bridge.StateFailed)
			} else {
				_ = a.db.UpdateMessageState(ctx, chat.String(), id, bridge.StateSent)
				a.log.Debug().Str("id", logx.Scrub(string(sentID))).Msg("sent")
			}
			a.reloadCurrent()
			return false
		})
	}()
}

func chatFor(ctx context.Context, db *store.DB, jid bridge.JID) bridge.Chat {
	c, err := db.GetChat(ctx, jid.String())
	if err != nil {
		return bridge.Chat{JID: jid, Title: jid.User}
	}
	return c
}

func parseJID(s string) bridge.JID {
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			return bridge.JID{User: s[:i], Server: s[i+1:]}
		}
	}
	return bridge.JID{User: s}
}
