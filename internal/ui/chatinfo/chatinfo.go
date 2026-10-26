// Package chatinfo renders the right-side chat-info drawer (WL-2.6):
// photo, about, members list for groups, mute/ephemeral/block
// actions. Group management (add/remove/promote/invites) and the
// safety-numbers dialog land at WL-4.1/4.6; the drawer already
// exposes the hooks.
package chatinfo

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// Callbacks wires drawer actions to the app shell.
type Callbacks struct {
	OnMute      func(chat bridge.JID, hours int64)
	OnBlock     func(chat bridge.JID, block bool)
	OnGroupEdit func(chat bridge.JID) // WL-4.1 member management
	OnBusiness  func(chat bridge.JID) // WL-4.5 business profile
	OnClose     func()                // header X: hide the drawer
}

// View is the chat-info drawer widget.
type View struct {
	Box     *gtk.Box
	heading *gtk.Label
	avatar  *gtk.Box
	title   *gtk.Label
	sub     *gtk.Label
	about   *gtk.Label
	aboutBx *gtk.Box
	members *gtk.Button
	cb      Callbacks
	chat    bridge.Chat
}

// New builds the drawer (hidden until the shell shows it).
func New(cb Callbacks) *View {
	v := &View{cb: cb}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	v.Box.AddCSSClass("info-drawer")
	v.Box.SetSizeRequest(380, -1)

	head := gtk.NewBox(gtk.OrientationHorizontal, 12)
	head.AddCSSClass("drawer-header")
	closeBtn := gtk.NewButtonFromIconName("wl-close-symbolic")
	closeBtn.AddCSSClass("icon-btn")
	closeBtn.SetVAlign(gtk.AlignCenter)
	closeBtn.SetTooltipText("Close")
	closeBtn.ConnectClicked(func() {
		if v.cb.OnClose != nil {
			v.cb.OnClose()
		}
	})
	head.Append(closeBtn)
	v.heading = gtk.NewLabel("Contact info")
	v.heading.AddCSSClass("drawer-title")
	head.Append(v.heading)
	v.Box.Append(head)

	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	body := gtk.NewBox(gtk.OrientationVertical, 0)
	scroll.SetChild(body)
	v.Box.Append(scroll)

	top := gtk.NewBox(gtk.OrientationVertical, 6)
	top.SetMarginTop(28)
	top.SetMarginBottom(20)
	v.avatar = gtk.NewBox(gtk.OrientationHorizontal, 0)
	v.avatar.SetHAlign(gtk.AlignCenter)
	v.avatar.SetMarginBottom(10)
	top.Append(v.avatar)
	v.title = gtk.NewLabel("")
	v.title.AddCSSClass("drawer-name")
	v.title.SetWrap(true)
	v.title.SetJustify(gtk.JustifyCenter)
	top.Append(v.title)
	v.sub = gtk.NewLabel("")
	v.sub.AddCSSClass("drawer-caption")
	top.Append(v.sub)
	body.Append(top)

	aboutBox := gtk.NewBox(gtk.OrientationVertical, 4)
	v.aboutBx = aboutBox
	aboutBox.AddCSSClass("drawer-section")
	cap := gtk.NewLabel("About")
	cap.AddCSSClass("drawer-caption")
	cap.SetXAlign(0)
	aboutBox.Append(cap)
	v.about = gtk.NewLabel("")
	v.about.SetXAlign(0)
	v.about.SetWrap(true)
	aboutBox.Append(v.about)
	body.Append(aboutBox)

	actions := gtk.NewBox(gtk.OrientationVertical, 0)
	actions.AddCSSClass("drawer-section")
	actions.SetMarginStart(0)
	actions.Append(action("wl-bell-off-symbolic", "Mute notifications for 8 hours", false, func() { v.cb.OnMute(v.chat.JID, 8) }))
	v.members = action("wl-people-symbolic", "Manage members", false, func() { v.cb.OnGroupEdit(v.chat.JID) })
	actions.Append(v.members)
	actions.Append(action("wl-person-symbolic", "Business profile", false, func() { v.cb.OnBusiness(v.chat.JID) }))
	actions.Append(action("wl-block-symbolic", "Block", true, func() { v.cb.OnBlock(v.chat.JID, true) }))
	body.Append(actions)
	return v
}

// action is one full-width drawer row with an icon.
func action(icon, text string, danger bool, onClick func()) *gtk.Button {
	b := gtk.NewButton()
	b.AddCSSClass("drawer-action")
	if danger {
		b.AddCSSClass("danger")
	}
	row := gtk.NewBox(gtk.OrientationHorizontal, 18)
	row.Append(gtk.NewImageFromIconName(icon))
	l := gtk.NewLabel(text)
	l.SetXAlign(0)
	row.Append(l)
	b.SetChild(row)
	b.ConnectClicked(onClick)
	return b
}

// ShowChat loads a chat into the drawer.
func (v *View) ShowChat(c bridge.Chat, contact bridge.Contact) {
	v.chat = c
	group := c.Kind == bridge.ChatGroup
	if child := v.avatar.FirstChild(); child != nil {
		v.avatar.Remove(child)
	}
	v.avatar.Append(widgets.AvatarFor(c.Title, group, 160))
	v.title.SetText(c.Title)
	about := contact.About
	if group {
		v.heading.SetText("Group info")
		v.sub.SetText("Group")
		if about == "" {
			about = "Group • members at WL-4.1"
		}
	} else {
		v.heading.SetText("Contact info")
		v.sub.SetText("+" + c.JID.User)
	}
	v.members.SetVisible(group)
	v.about.SetText(about)
	v.aboutBx.SetVisible(about != "")
}
