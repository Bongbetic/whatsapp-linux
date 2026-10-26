// Package app owns the Gtk.Application lifecycle (WL-2.1): single
// instance, main window (1180×760, header bar with connection dot,
// two-pane Paned), login↔main view swap, event dispatcher (§8: core
// channel → glib.IdleAdd), global shortcuts, close-to-tray.
package app

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/rs/zerolog"

	"github.com/Bongbetic/whatsapp-linux/internal/autostart"
	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/core"
	"github.com/Bongbetic/whatsapp-linux/internal/links"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
	"github.com/Bongbetic/whatsapp-linux/internal/media"
	"github.com/Bongbetic/whatsapp-linux/internal/notify"
	"github.com/Bongbetic/whatsapp-linux/internal/settings"
	"github.com/Bongbetic/whatsapp-linux/internal/store"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/business"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/call"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/calls"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/channels"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/chatinfo"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/chatlist"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/convo"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/dialogs"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/groups"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/login"
	uisettings "github.com/Bongbetic/whatsapp-linux/internal/ui/settings"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/status"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/theme"
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
	links    []links.Intent
	linkCh   <-chan string
	shot     *ShotConfig // dev-only screenshot mode (shot.go)

	dataDir  string
	cacheDir string

	win       *gtk.ApplicationWindow
	connLbl   *gtk.Label
	banner    *gtk.Revealer
	bannerL   *gtk.Label
	stack     *gtk.Stack
	paned     *gtk.Paned
	rail      *gtk.Box
	railBtn   map[string]*gtk.Button
	railBadge *gtk.Label
	voice     *media.Recorder
	loginV    *login.View
	chatsV    *chatlist.View
	convV     *convo.View
	infoV     *chatinfo.View
	statusV   *status.View
	chansV    *channels.View
	callsV    *calls.View
	callWin   *call.Window
	callID    string
	ringID    string
	ringStop  chan struct{}
	notif     *notify.Notifier

	narrow     bool // single-pane layout under narrowWidth
	narrowList bool // narrow mode currently shows the list (not convo)

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
	DataDir  string // ~/.local/share/whatsapp-linux (§6)
	CacheDir string // ~/.cache/whatsapp-linux (§6)
	// Links are startup URI intents (argv); LinkCh carries
	// single-instance forwards from later invocations (single.go).
	Links  []links.Intent
	LinkCh <-chan string
	// Shot enables dev-only screenshot mode (cmd/wlshot).
	Shot *ShotConfig
}

// Run starts the application (blocks until quit). No GTK calls happen
// before this, so headless commands can build Config without a display.
func Run(cfg Config) int {
	a := &App{engine: cfg.Engine, db: cfg.DB, settings: cfg.Settings, log: cfg.Log, version: cfg.Version, hidden: cfg.Hidden, dataDir: cfg.DataDir, cacheDir: cfg.CacheDir, links: cfg.Links, linkCh: cfg.LinkCh, shot: cfg.Shot}
	a.gtkApp = gtk.NewApplication(bridge.AppID, appFlags(cfg))
	a.gtkApp.ConnectActivate(a.activate)
	return a.gtkApp.Run(nil)
}

func (a *App) activate() {
	a.shotDark()
	a.win = gtk.NewApplicationWindow(a.gtkApp)
	a.win.SetTitle(bridge.DisplayName)
	a.win.SetDefaultSize(a.settings.GetInt(settings.KeyWindowW), a.settings.GetInt(settings.KeyWindowH))
	theme.Install(gdk.DisplayGetDefault(), a.cacheDir, a.log)

	// Windows-style caption: app icon + name on the left, flat
	// minimize/maximize/close on the right, no centered title.
	bar := gtk.NewHeaderBar()
	bar.SetDecorationLayout(":minimize,maximize,close")
	appIcon := gtk.NewImageFromIconName(bridge.AppID)
	appIcon.AddCSSClass("app-icon")
	bar.PackStart(appIcon)
	title := gtk.NewLabel(bridge.DisplayName)
	title.AddCSSClass("app-title")
	bar.PackStart(title)
	a.connLbl = gtk.NewLabel("●")
	a.connLbl.AddCSSClass("conn-dot")
	a.connLbl.SetTooltipText("Connecting…")
	bar.PackStart(a.connLbl)
	bar.SetTitleWidget(gtk.NewBox(gtk.OrientationHorizontal, 0))
	a.win.SetTitlebar(bar)

	// Rail on the left at full height; the banner sits above the
	// page area, like the reference client's chat-list butterbar.
	root := gtk.NewBox(gtk.OrientationHorizontal, 0)
	a.buildRail()
	root.Append(a.rail)
	column := gtk.NewBox(gtk.OrientationVertical, 0)
	column.SetHExpand(true)
	// The content sits in a card inset from the caption bar and rail,
	// with a rounded top-left corner like the reference client.
	column.AddCSSClass("content-card")
	column.SetOverflow(gtk.OverflowHidden)
	root.Append(column)
	a.banner, a.bannerL = widgets.Banner()
	column.Append(a.banner)

	a.stack = gtk.NewStack()
	a.stack.SetHExpand(true)
	a.stack.SetVExpand(true)
	column.Append(a.stack)
	main := gtk.NewBox(gtk.OrientationVertical, 0)
	a.paned = gtk.NewPaned(gtk.OrientationHorizontal)
	a.paned.SetVExpand(true)
	a.paned.SetWideHandle(false)
	a.paned.SetResizeStartChild(false)
	a.paned.SetShrinkStartChild(false)
	a.paned.SetShrinkEndChild(false)
	a.paned.SetPosition(chatListWidth)

	a.chatsV = chatlist.New(a.openChat, func() { dialogs.NewChat(a.win, a.newChatFromPhone) })
	a.chatsV.SetPreview(a.chatPreview)
	a.chatsV.Box.SetSizeRequest(320, -1)
	a.paned.SetStartChild(a.chatsV.Box)

	right := gtk.NewBox(gtk.OrientationHorizontal, 0)
	a.convV = convo.New(convo.Callbacks{
		OnSend:   a.send,
		OnTyping: func(chat bridge.JID, typing bool) { _ = a.engine.SetTyping(context.Background(), chat, typing) },
		OnDraft: func(chat bridge.JID, text string) {
			_ = a.db.UpsertChat(context.Background(), bridge.Chat{JID: chat, Draft: text})
		},
		OnAttach: func(chat bridge.JID) { a.attach(chat) },
		OnCall:   func(chat bridge.JID, video bool) { a.placeCall(chat, video) },
		OnBack:   func() { a.showListNarrow() },
		OnMic:    func() { a.toggleRecord() },
		OnInfo:   func() { a.infoV.Box.SetVisible(!a.infoV.Box.Visible()) },
		NameFor:  a.displayName,
	}, a.settings.GetBool(settings.KeyEnterToSend))
	right.Append(a.convV.Box)
	a.convV.Box.SetHExpand(true)
	a.infoV = chatinfo.New(chatinfo.Callbacks{
		OnMute:      func(chat bridge.JID, h int64) { a.toggleMute(chat, h) },
		OnBlock:     func(chat bridge.JID, b bool) { _ = a.engine.BlockSet(context.Background(), chat, b) },
		OnGroupEdit: func(chat bridge.JID) { groups.Manage(a.win, a.engine, chat) },
		OnBusiness:  func(chat bridge.JID) { business.Show(a.win, a.engine, chat) },
		OnClose:     func() { a.infoV.Box.SetVisible(false) },
	})
	a.infoV.Box.SetVisible(false)
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

	settingsPage := uisettings.New(a.settings, func() {
		a.stack.SetVisibleChildName("privacy")
	})
	settingsPage.Box.Append(uisettings.StorageSection(a.dataDir, a.cacheDir))
	a.stack.AddNamed(sidePage("Settings", "wl-settings-symbolic", "Notifications, privacy, storage and startup.", settingsPage.Box), "settings")
	a.stack.AddNamed(sidePage("Privacy", "wl-lock-symbolic", "Choose who can see your info, and manage blocked contacts.", uisettings.PrivacySection(a.engine, a.log)), "privacy")
	a.statusV = status.New(a.db, a.engine, func(to bridge.JID, text string) {
		_, _ = a.engine.SendText(context.Background(), to, text, bridge.SendOpts{})
	})
	a.stack.AddNamed(sidePage("Status", "wl-status-symbolic", "Share photos, videos and text that disappear after 24 hours.", a.statusV.Box), "status")
	a.chansV = channels.New(a.db, a.engine)
	a.stack.AddNamed(sidePage("Channels", "wl-channels-symbolic", "Follow channels to get updates on topics you care about.", a.chansV.Box), "channels")
	a.callsV = calls.New(a.db, a.engine, a.displayName, func(chat bridge.JID, video bool) { a.placeCall(chat, video) })
	a.stack.AddNamed(sidePage("Calls", "wl-calls-symbolic", "Make voice and video calls to your contacts.", a.callsV.Box), "calls")

	a.win.SetChild(root)

	a.notif = notify.New(a.gtkApp, a.settings, a.log, notify.TrayFor(a.win),
		func(chatJID, text string) {
			if _, err := a.engine.SendText(context.Background(), parseJID(chatJID), text, bridge.SendOpts{}); err != nil {
				a.log.Error().Err(err).Msg("inline reply failed")
				widgets.SetBanner(a.banner, a.bannerL, "Reply failed: "+err.Error())
			}
		},
		func(chatJID string) { _ = a.engine.MarkRead(context.Background(), parseJID(chatJID)) },
		func(chatJID string) {
			glib.IdleAdd(func() bool {
				a.openChatByJID(parseJID(chatJID))
				return false
			})
		},
	)

	a.win.ConnectCloseRequest(func() bool {
		a.settings.Set(settings.KeyWindowW, a.win.Width())
		a.settings.Set(settings.KeyWindowH, a.win.Height())
		if a.settings.GetBool(settings.KeyCloseToTray) {
			a.win.SetVisible(false) // keep running in tray (WL-2.5)
			return true
		}
		return false
	})
	a.shortcuts()
	glib.TimeoutAdd(500, func() bool {
		a.applyNarrow()
		return true
	})

	if !a.hidden {
		a.win.Show()
	}
	_ = autostart.Set(a.settings.GetBool(settings.KeyAutostart))
	a.showLogin("Connecting…")
	go a.dispatch()
	if err := a.engine.Start(context.Background()); err != nil {
		a.log.Error().Err(err).Msg("engine start")
	}
	go a.drainLinks()
	a.shotSchedule()
}

// drainLinks opens startup URI intents, then serves single-instance
// forwards for the life of the process.
func (a *App) drainLinks() {
	for _, in := range a.links {
		in := in
		glib.IdleAdd(func() bool {
			a.openLinkIntent(in)
			return false
		})
	}
	if a.linkCh == nil {
		return
	}
	for raw := range a.linkCh {
		raw := raw
		glib.IdleAdd(func() bool {
			in, err := links.Parse(raw)
			if err != nil {
				a.log.Error().Err(err).Msg("bad forwarded link")
				return false
			}
			a.openLinkIntent(in)
			return false
		})
	}
}

// shortcuts wires the §9 global keys.
func (a *App) shortcuts() {
	keys := gtk.NewEventControllerKey()
	keys.ConnectKeyPressed(func(keyval, _ uint, state gdk.ModifierType) bool {
		if state&gdk.ControlMask == 0 {
			return false
		}
		shift := state&gdk.ShiftMask != 0
		switch keyval {
		case gdk.KEY_n, gdk.KEY_N:
			if !shift {
				dialogs.NewChat(a.win, a.newChatFromPhone)
				return true
			}
		case gdk.KEY_f, gdk.KEY_F:
			if shift {
				a.showPage("main")
				a.convV.ToggleSearch(true)
			} else {
				a.showPage("main")
				a.chatsV.FocusSearch()
			}
			return true
		case gdk.KEY_m, gdk.KEY_M:
			if shift && a.hasChat {
				a.toggleMute(a.current.JID, 8)
				return true
			}
		case gdk.KEY_q, gdk.KEY_Q:
			if !shift {
				a.win.SetVisible(false) // quit-to-tray
				return true
			}
		case gdk.KEY_comma:
			a.showPage("settings")
			return true
		case gdk.KEY_Escape:
			a.chatsV.ClearSearch()
			a.showPage("main")
			return true
		}
		return false
	})
	a.win.AddController(keys)
}

// newChatFromPhone starts (or opens) a DM with a typed phone number.
func (a *App) newChatFromPhone(phone string) {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	if len(digits) < 7 {
		widgets.SetBanner(a.banner, a.bannerL, "Enter a phone number with at least 7 digits")
		return
	}
	a.openChatByJID(bridge.JID{User: digits, Server: "s.whatsapp.net"})
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
	case bridge.CallEvent:
		a.onCall(e)
	case bridge.RefreshEvent:
		a.refreshChats()
		a.reloadCurrent()
	}
}

// placeCall dials out via the fork (WL-5.1) and opens the call window.
func (a *App) placeCall(chat bridge.JID, video bool) {
	id, err := a.engine.CallPlace(context.Background(), chat, video)
	if err != nil {
		widgets.SetBanner(a.banner, a.bannerL, "Call failed: "+err.Error())
		return
	}
	a.callID = id
	a.callWin = call.Show(a.win, bridge.CallEvent{ID: id, Direction: "out", Media: map[bool]string{true: "video", false: "audio"}[video], Stage: "offering"},
		func() { _ = a.engine.CallHangup(context.Background(), id) },
		func(muted bool) { a.engine.SetCallMuted(id, muted) })
}

// onCall routes incoming switch transitions to the call UI (WL-5.3).
func (a *App) onCall(e bridge.CallEvent) {
	if e.Direction == "in" && e.Stage == "ringing" {
		a.callID = e.ID
		a.playRingtone()
		call.Incoming(a.win, e,
			func() {
				a.stopRingtone()
				if err := a.engine.CallAccept(context.Background(), e.ID, e.Media == "video"); err != nil {
					widgets.SetBanner(a.banner, a.bannerL, "Accept failed: "+err.Error())
				}
			},
			func() {
				a.stopRingtone()
				_ = a.engine.CallReject(context.Background(), e.Chat, e.ID)
			})
		return
	}
	if e.ID == a.callID && a.callWin != nil {
		if e.Stage == "ended" {
			a.callWin.Close()
			a.callWin = nil
			a.callsV.Refresh()
			return
		}
		a.callWin.SetStage(e.Stage)
	}
	if e.Stage == "ended" && e.ID == a.ringID {
		a.stopRingtone()
	}
}

// playRingtone loops the ringtone until stopRingtone (accept,
// reject, or the remote end hanging up).
func (a *App) playRingtone() {
	a.stopRingtone()
	stop := make(chan struct{})
	a.ringID, a.ringStop = a.callID, stop
	path := notify.TonePath("ringtone.wav")
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			notify.PlayFile(path)
			select {
			case <-stop:
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
}

func (a *App) stopRingtone() {
	if a.ringStop != nil {
		select {
		case <-a.ringStop:
		default:
			close(a.ringStop)
		}
		a.ringStop = nil
		a.ringID = ""
	}
}

func progressText(p int) string { return "Syncing history… " + strconv.Itoa(p) + "%" }

func (a *App) onConn(s bridge.ConnState) {
	switch s {
	case bridge.ConnConnected:
		a.connLbl.RemoveCSSClass("offline")
		a.connLbl.SetTooltipText("Connected")
		widgets.SetBanner(a.banner, a.bannerL, "")
		a.notif.SetConnected(true)
		a.showMain()
	case bridge.ConnConnecting:
		a.connLbl.SetTooltipText("Connecting…")
		widgets.SetBanner(a.banner, a.bannerL, "Connecting…")
		a.notif.SetConnected(false)
	case bridge.ConnOffline:
		a.connLbl.AddCSSClass("offline")
		a.connLbl.SetTooltipText("Offline — retrying")
		widgets.SetBanner(a.banner, a.bannerL, "Offline — retrying…")
		a.notif.SetConnected(false)
	case bridge.ConnLoggedOut:
		a.notif.SetConnected(false)
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
	a.rail.SetVisible(false)
	a.stack.SetVisibleChildName("login")
}

func (a *App) showMain() {
	a.refreshChats()
	a.rail.SetVisible(true)
	a.showPage("main")
}

// railItem is one navigation-rail destination; the filled icon
// marks the active page, like the reference client.
type railItem struct {
	page, icon, tip string
}

var (
	railTop = []railItem{
		{"main", "wl-chats", "Chats"},
		{"calls", "wl-calls", "Calls"},
		{"status", "wl-status", "Status"},
		{"channels", "wl-channels", "Channels"},
	}
	railBottom = []railItem{
		{"settings", "wl-settings", "Settings"},
	}
)

// buildRail creates the Windows-style icon nav: chats, calls,
// status, channels on top; settings and profile at the bottom.
func (a *App) buildRail() {
	a.rail = gtk.NewBox(gtk.OrientationVertical, 6)
	a.rail.AddCSSClass("rail")
	a.railBtn = map[string]*gtk.Button{}
	add := func(it railItem) {
		b := gtk.NewButtonFromIconName(it.icon + "-symbolic")
		b.SetTooltipText(it.tip)
		b.AddCSSClass("rail-btn")
		b.ConnectClicked(func() { a.showPage(it.page) })
		a.railBtn[it.page] = b
		if it.page != "main" {
			a.rail.Append(b)
			return
		}
		// Chats carries the unread-chats badge.
		ov := gtk.NewOverlay()
		ov.SetChild(b)
		a.railBadge = gtk.NewLabel("")
		a.railBadge.AddCSSClass("rail-badge")
		a.railBadge.SetHAlign(gtk.AlignEnd)
		a.railBadge.SetVAlign(gtk.AlignStart)
		a.railBadge.SetCanTarget(false)
		a.railBadge.SetVisible(false)
		ov.AddOverlay(a.railBadge)
		a.rail.Append(ov)
	}
	for _, it := range railTop {
		add(it)
	}
	spacer := gtk.NewBox(gtk.OrientationVertical, 0)
	spacer.SetVExpand(true)
	a.rail.Append(spacer)
	sep := gtk.NewBox(gtk.OrientationHorizontal, 0)
	sep.AddCSSClass("rail-sep")
	a.rail.Append(sep)
	for _, it := range railBottom {
		add(it)
	}
	profBtn := gtk.NewButton()
	profBtn.SetChild(widgets.AvatarFor("Profile", false, 28))
	profBtn.SetTooltipText("Profile")
	profBtn.AddCSSClass("rail-btn")
	profBtn.ConnectClicked(func() { a.showPage("settings") })
	a.rail.Append(profBtn)
}

// sidePage lays a page out like the reference client's non-chat
// tabs: a list panel at the chat-list width on the left and an
// intro placeholder filling the rest.
func sidePage(title, icon, blurb string, content gtk.Widgetter) *gtk.Box {
	page := gtk.NewBox(gtk.OrientationHorizontal, 0)
	panel := gtk.NewBox(gtk.OrientationVertical, 0)
	panel.AddCSSClass("side-panel")
	panel.SetSizeRequest(chatListWidth, -1)
	// Explicit, so expanding rows inside never widen the panel.
	panel.SetHExpand(false)
	head := gtk.NewBox(gtk.OrientationHorizontal, 0)
	head.AddCSSClass("pane-header")
	t := gtk.NewLabel(title)
	t.AddCSSClass("pane-title")
	t.SetXAlign(0)
	head.Append(t)
	panel.Append(head)
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	body := gtk.NewBox(gtk.OrientationVertical, 0)
	body.AddCSSClass("page-content")
	body.Append(content)
	scroll.SetChild(body)
	panel.Append(scroll)
	page.Append(panel)
	sep := gtk.NewSeparator(gtk.OrientationVertical)
	page.Append(sep)

	intro := gtk.NewBox(gtk.OrientationVertical, 8)
	intro.AddCSSClass("intro-panel")
	intro.SetHExpand(true)
	mid := gtk.NewBox(gtk.OrientationVertical, 8)
	mid.SetVExpand(true)
	mid.SetVAlign(gtk.AlignCenter)
	img := gtk.NewImageFromIconName(icon)
	img.AddCSSClass("intro-icon")
	mid.Append(img)
	it := gtk.NewLabel(title)
	it.AddCSSClass("intro-title")
	mid.Append(it)
	txt := gtk.NewLabel(blurb)
	txt.AddCSSClass("intro-text")
	txt.SetWrap(true)
	txt.SetJustify(gtk.JustifyCenter)
	mid.Append(txt)
	intro.Append(mid)
	page.Append(intro)
	return page
}

// showPage switches the stack and highlights the rail (filled icon
// on the active destination).
func (a *App) showPage(page string) {
	for _, list := range [][]railItem{railTop, railBottom} {
		for _, it := range list {
			b := a.railBtn[it.page]
			if it.page == page || (page == "privacy" && it.page == "settings") {
				b.AddCSSClass("active")
				b.SetIconName(it.icon + "-filled-symbolic")
			} else {
				b.RemoveCSSClass("active")
				b.SetIconName(it.icon + "-symbolic")
			}
		}
	}
	switch page {
	case "status":
		a.statusV.Refresh()
	case "channels":
		a.chansV.Refresh()
	case "calls":
		a.callsV.Refresh()
	}
	a.stack.SetVisibleChildName(page)
}

// displayName resolves any JID to contact name, push name, or phone.
func (a *App) displayName(jid bridge.JID) string {
	if c, err := a.db.GetContact(context.Background(), jid.String()); err == nil {
		if n := core.DisplayName(c, ""); n != "" {
			return n
		}
	}
	if jid.Server == "s.whatsapp.net" || jid.Server == "c.us" {
		return "+" + jid.User
	}
	return jid.String()
}

// chatPreview renders one list row's snippet from the last message.
func (a *App) chatPreview(c bridge.Chat) string {
	m, err := a.db.LastMessage(context.Background(), c.JID.String())
	if err != nil {
		return ""
	}
	sender := ""
	if !m.IsFromMe && c.Kind == bridge.ChatGroup {
		sender = a.displayName(m.Sender)
	}
	return widgets.ChatPreview(m, sender)
}

// toggleRecord starts/stops a voice-note recording; stopping sends it.
func (a *App) toggleRecord() {
	if !a.hasChat {
		return
	}
	if a.voice != nil {
		path, err := a.voice.Stop()
		a.voice = nil
		a.convV.SetMicRecording(false)
		if err != nil {
			widgets.SetBanner(a.banner, a.bannerL, "Recording failed: "+err.Error())
			return
		}
		go a.sendVoice(a.current.JID, path)
		return
	}
	r := &media.Recorder{}
	dst := filepath.Join(a.cacheDir, "voice-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".ogg")
	if err := r.Start(dst); err != nil {
		widgets.SetBanner(a.banner, a.bannerL, "Mic unavailable: "+err.Error())
		return
	}
	a.voice = r
	a.convV.SetMicRecording(true)
	widgets.SetBanner(a.banner, a.bannerL, "🔴 Recording voice — tap mic to stop & send")
}

// sendVoice uploads a finished recording as a voice message.
func (a *App) sendVoice(chat bridge.JID, path string) {
	ctx := context.Background()
	id, err := a.engine.SendMedia(ctx, chat, bridge.MediaFile{Path: path, MIME: "audio/ogg", Kind: "voice"}, bridge.SendOpts{})
	glib.IdleAdd(func() bool {
		widgets.SetBanner(a.banner, a.bannerL, "")
		if err != nil {
			a.log.Error().Err(err).Msg("voice send failed")
			widgets.SetBanner(a.banner, a.bannerL, "Voice send failed: "+err.Error())
			return false
		}
		a.log.Debug().Str("id", logx.Scrub(string(id))).Msg("voice sent")
		a.reloadCurrent()
		return false
	})
}

func (a *App) refreshChats() {
	chats, err := a.db.ListChats(context.Background(), 200, 0, "", false)
	if err != nil {
		a.log.Error().Err(err).Msg("list chats")
		return
	}
	// Resolve DM titles against contacts so rows created before a
	// name was learned (bare JID/phone) display the name immediately.
	ctx := context.Background()
	for i, c := range chats {
		if c.Kind != bridge.ChatDM {
			continue
		}
		if contact, err := a.db.GetContact(ctx, c.JID.String()); err == nil {
			if n := core.DisplayName(contact, ""); n != "" {
				chats[i].Title = n
			}
		}
		if chats[i].Title == "" {
			chats[i].Title = c.JID.User
		}
	}
	a.chatsV.SetChats(chats)
	if n := a.chatsV.UnreadChats(); n > 0 {
		a.railBadge.SetText(strconv.Itoa(n))
		a.railBadge.SetVisible(true)
	} else {
		a.railBadge.SetVisible(false)
	}
}

// narrowWidth is the §9 single-pane breakpoint.
const narrowWidth = 700

// chatListWidth is the chat-list pane's initial width (reference
// client at its default window size).
const chatListWidth = 400

// applyNarrow collapses to one pane under narrowWidth (Back button
// returns to the list) and restores both panes above it.
func (a *App) applyNarrow() {
	if a.win == nil {
		return
	}
	narrow := a.win.Width() >= 0 && a.win.Width() < narrowWidth
	if narrow == a.narrow && a.win.Width() != 0 {
		a.layoutPanes()
		return
	}
	a.narrow = narrow
	a.layoutPanes()
}

// layoutPanes shows list/convo per narrow state.
func (a *App) layoutPanes() {
	if !a.narrow {
		a.chatsV.Box.SetVisible(true)
		a.convV.Box.SetVisible(true)
		a.convV.SetBackVisible(false)
		a.narrowList = false
		return
	}
	a.convV.SetBackVisible(true)
	if a.narrowList || !a.hasChat {
		a.chatsV.Box.SetVisible(true)
		a.convV.Box.SetVisible(false)
		return
	}
	a.chatsV.Box.SetVisible(false)
	a.convV.Box.SetVisible(true)
}

// showListNarrow returns to the list in narrow mode (Back button).
func (a *App) showListNarrow() {
	a.narrowList = true
	a.layoutPanes()
}

func (a *App) openChat(c bridge.Chat) {
	a.showPage("main")
	a.current, a.hasChat, a.narrowList = c, true, false
	if a.narrow {
		a.layoutPanes()
	}
	msgs, err := a.db.ListMessages(context.Background(), c.JID.String(), 50, 0, "")
	if err != nil {
		a.log.Error().Err(err).Msg("list messages")
	}
	// ListMessages is newest-first; reverse for display.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	a.convV.OpenChat(c, msgs)
	a.chatsV.SetSelected(c.JID.String())
	contact, _ := a.db.GetContact(context.Background(), c.JID.String())
	a.infoV.ShowChat(c, contact)
	a.notif.Clear()
}

// openChatByJID opens a chat from a JID string (notification actions,
// single-instance URI handoff): unknown chats get a row so the
// conversation can start immediately.
func (a *App) openChatByJID(jid bridge.JID) {
	ctx := context.Background()
	c, err := a.db.GetChat(ctx, jid.String())
	if err != nil {
		title := jid.User
		if contact, cerr := a.db.GetContact(ctx, jid.String()); cerr == nil {
			if n := core.DisplayName(contact, ""); n != "" {
				title = n
			}
		}
		c = bridge.Chat{JID: jid, Kind: bridge.ChatDM, Title: title}
		if uerr := a.db.UpsertChat(ctx, c); uerr != nil {
			a.log.Error().Err(uerr).Msg("create chat row")
			return
		}
		a.refreshChats()
	}
	a.win.Show()
	a.openChat(c)
}

// openLinkIntent handles an inbound whatsapp://, wa.me or tel: link
// (startup arg or single-instance forward): open the DM and seed the
// composer with any prefilled text.
func (a *App) openLinkIntent(in links.Intent) {
	jid := bridge.JID{User: in.Phone, Server: "s.whatsapp.net"}
	ctx := context.Background()
	c, err := a.db.GetChat(ctx, jid.String())
	if err != nil {
		title := in.Phone
		if contact, cerr := a.db.GetContact(ctx, jid.String()); cerr == nil {
			if n := core.DisplayName(contact, ""); n != "" {
				title = n
			}
		}
		c = bridge.Chat{JID: jid, Kind: bridge.ChatDM, Title: title, Draft: in.Text}
		if uerr := a.db.UpsertChat(ctx, c); uerr != nil {
			a.log.Error().Err(uerr).Msg("create link chat row")
			return
		}
		a.refreshChats()
	} else if in.Text != "" {
		c.Draft = in.Text
		_ = a.db.UpsertChat(ctx, c)
	}
	a.win.Show()
	a.openChat(c)
}

// toggleMute mutes (or unmutes an already-muted chat) via app-state.
func (a *App) toggleMute(chat bridge.JID, hours int64) {
	ctx := context.Background()
	muted := true
	if c, err := a.db.GetChat(ctx, chat.String()); err == nil {
		if c.MutedUntil == -1 || c.MutedUntil > time.Now().Unix() {
			muted = false // already muted → unmute
		}
	}
	if err := a.engine.SetMuted(ctx, chat, muted, hours); err != nil {
		a.log.Error().Err(err).Msg("mute failed")
		widgets.SetBanner(a.banner, a.bannerL, "Mute failed: "+err.Error())
	}
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

// attach implements the WL-2.3/WL-3.x attachment path: native file
// picker → prepare (image cap / video transcode) → SendMedia with
// the composer draft as caption.
func (a *App) attach(chat bridge.JID) {
	dlg := gtk.NewFileChooserNative("Attach file", &a.win.Window, gtk.FileChooserActionOpen, "Send", "Cancel")
	dlg.ConnectResponse(func(id int) {
		defer dlg.Destroy()
		if id != int(gtk.ResponseAccept) || dlg.File() == nil {
			return
		}
		path := dlg.File().Path()
		go a.sendFile(chat, path)
	})
	dlg.Show()
}

func (a *App) sendFile(chat bridge.JID, path string) {
	ctx := context.Background()
	kind, mime := classifyFile(path)
	final := path
	var err error
	switch kind {
	case "image":
		final, err = media.PrepareImage(path, a.cacheDir, false)
	case "video":
		if need, perr := media.NeedsTranscode(path); perr == nil && need {
			out := final + ".conv.mp4"
			if terr := media.TranscodeMP4(path, out); terr == nil {
				final = out
			} else {
				err = terr
			}
		} else if perr != nil {
			err = perr
		}
	}
	if err != nil {
		a.log.Error().Err(err).Msg("prepare attachment")
		return
	}
	id, err := a.engine.SendMedia(ctx, chat, bridge.MediaFile{Path: final, MIME: mime, Kind: kind}, bridge.SendOpts{})
	glib.IdleAdd(func() bool {
		if err != nil {
			a.log.Error().Err(err).Msg("send media failed")
			return false
		}
		a.log.Debug().Str("id", logx.Scrub(string(id))).Msg("media sent")
		a.reloadCurrent()
		return false
	})
}

// classifyFile maps a picked path to (media kind, MIME) by extension.
func classifyFile(path string) (string, string) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image", "image/jpeg"
	case ".png":
		return "image", "image/png"
	case ".webp":
		return "image", "image/webp"
	case ".mp4", ".mov", ".mkv", ".webm":
		return "video", "video/mp4"
	case ".ogg", ".oga":
		return "audio", "audio/ogg"
	case ".mp3":
		return "audio", "audio/mpeg"
	default:
		return "doc", "application/octet-stream"
	}
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
