package notify

import (
	"fmt"
	"os"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/godbus/dbus/v5"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// trayIcon is the installed app icon name hosts render.
const trayIcon = "com.bongbetic.WhatsAppLinux"

// TrayFor returns a StatusNotifier tray when a session bus and a
// watcher host exist, else a NoopTray (headless/CI, WL-2.5).
func TrayFor(win *gtk.ApplicationWindow) Tray {
	t, err := newSNITray(win)
	if err != nil {
		return NoopTray{}
	}
	return t
}

// SNITray publishes org.kde.StatusNotifierItem so desktop hosts
// (GNOME + extension, KDE, trayer) show our icon, unread state and
// activation. Best-effort: any D-Bus failure leaves NoopTray.
type SNITray struct {
	conn   *dbus.Conn
	path   dbus.ObjectPath
	show   func()
	unread int
	online bool
}

// newSNITray registers the item; error means "use NoopTray".
func newSNITray(win *gtk.ApplicationWindow) (*SNITray, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	if !nameOwned(conn, "org.kde.StatusNotifierWatcher") {
		conn.Close()
		return nil, fmt.Errorf("no StatusNotifierWatcher host")
	}
	t := &SNITray{
		conn: conn,
		path: "/StatusNotifierItem",
		show: func() {
			glib.IdleAdd(func() bool {
				win.Show()
				win.GrabFocus()
				return false
			})
		},
	}
	svc := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	if reply, err := conn.RequestName(svc, dbus.NameFlagDoNotQueue); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		return nil, fmt.Errorf("name %s unavailable", svc)
	}
	if err := conn.Export(t, t.path, "org.kde.StatusNotifierItem"); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.Export(props{t}, t.path, "org.freedesktop.DBus.Properties"); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher").
		Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, svc).Err; err != nil {
		conn.Close()
		return nil, err
	}
	return t, nil
}

func nameOwned(conn *dbus.Conn, name string) bool {
	var owned bool
	_ = conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus").
		Call("org.freedesktop.DBus.NameHasOwner", 0, name).Store(&owned)
	return owned
}

// props adapts SNITray to org.freedesktop.DBus.Properties.
type props struct{ t *SNITray }

func (p props) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	all, err := p.GetAll(iface)
	if err != nil {
		return dbus.Variant{}, err
	}
	v, ok := all[prop]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{prop})
	}
	return v, nil
}

func (p props) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	if iface != "org.kde.StatusNotifierItem" {
		return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	return p.t.propMap(), nil
}

func (p props) Set(iface, prop string, v dbus.Variant) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{prop})
}

func (t *SNITray) propMap() map[string]dbus.Variant {
	status := "Active"
	if t.unread > 0 {
		status = "NeedsAttention"
	}
	title := bridge.DisplayName
	tip := "WhatsApp (unofficial)"
	if t.unread > 0 {
		tip = fmt.Sprintf("%d unread message(s)", t.unread)
	}
	return map[string]dbus.Variant{
		"Category":          dbus.MakeVariant("ApplicationStatus"),
		"Id":                dbus.MakeVariant(bridge.BinaryName),
		"Title":             dbus.MakeVariant(title),
		"Status":            dbus.MakeVariant(status),
		"WindowId":          dbus.MakeVariant(int32(0)),
		"IconName":          dbus.MakeVariant(trayIcon),
		"AttentionIconName": dbus.MakeVariant(trayIcon),
		"ToolTip":           dbus.MakeVariant([]any{"", []any{0}, tip, ""}),
	}
}

// Activate shows the window (left-click on the tray icon).
func (t *SNITray) Activate(x, y int32) *dbus.Error {
	t.show()
	return nil
}

// ContextMenu shows the window (no menu yet; same as activate).
func (t *SNITray) ContextMenu(x, y int32) *dbus.Error {
	t.show()
	return nil
}

// SecondaryActivate mirrors Activate.
func (t *SNITray) SecondaryActivate(x, y int32) *dbus.Error {
	t.show()
	return nil
}

func (t *SNITray) Scroll(delta int32, dir string) *dbus.Error { return nil }

// SetUnread updates the attention state + tooltip.
func (t *SNITray) SetUnread(n int) {
	t.unread = n
	t.emitStatus()
}

// SetConnected tracks online state (banner owns the detail).
func (t *SNITray) SetConnected(online bool) { t.online = online }

func (t *SNITray) emitStatus() {
	_ = t.conn.Emit(t.path, "org.kde.StatusNotifierItem.NewStatus", t.status())
	_ = t.conn.Emit(t.path, "org.kde.StatusNotifierItem.NewToolTip")
}

func (t *SNITray) status() string {
	if t.unread > 0 {
		return "NeedsAttention"
	}
	return "Active"
}
