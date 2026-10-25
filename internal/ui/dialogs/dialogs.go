// Package dialogs holds small modal dialogs (§9): new chat (contact
// picker + message-a-number), new group (WL-4.1 hook), forward
// picker, delete/revoke confirm, logout, about.
package dialogs

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// NewChat asks for a phone number / JID to message.
func NewChat(parent *gtk.ApplicationWindow, onPick func(phone string)) {
	d := gtk.NewDialog()
	d.SetTransientFor(&parent.Window)
	d.SetTitle("New chat")
	box := gtk.NewBox(gtk.OrientationVertical, 8)
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("Phone number")
	box.Append(entry)
	go_ := gtk.NewButtonWithLabel("Message")
	go_.ConnectClicked(func() {
		onPick(entry.Buffer().Text())
		d.Close()
	})
	box.Append(go_)
	d.SetChild(box)
	d.Show()
}

// ConfirmDelete asks whether to delete or revoke a message.
func ConfirmDelete(parent *gtk.ApplicationWindow, onChoice func(revoke bool)) {
	d := gtk.NewDialog()
	d.SetTransientFor(&parent.Window)
	d.SetTitle("Delete message")
	box := gtk.NewBox(gtk.OrientationVertical, 8)
	del := gtk.NewButtonWithLabel("Delete for me")
	del.ConnectClicked(func() { onChoice(false); d.Close() })
	box.Append(del)
	rev := gtk.NewButtonWithLabel("Delete for everyone")
	rev.ConnectClicked(func() { onChoice(true); d.Close() })
	box.Append(rev)
	d.SetChild(box)
	d.Show()
}

// ForwardPicker picks a target chat to forward into (WL-2.4).
func ForwardPicker(parent *gtk.ApplicationWindow, chats []bridge.Chat, onPick func(bridge.Chat)) {
	d := gtk.NewDialog()
	d.SetTransientFor(&parent.Window)
	d.SetTitle("Forward to…")
	list := gtk.NewListBox()
	for _, c := range chats {
		c := c
		row := gtk.NewButtonWithLabel(c.Title)
		row.ConnectClicked(func() { onPick(c); d.Close() })
		list.Append(row)
	}
	scroll := gtk.NewScrolledWindow()
	scroll.SetChild(list)
	scroll.SetMinContentHeight(300)
	d.SetChild(scroll)
	d.Show()
}

// About shows the about dialog.
func About(parent *gtk.ApplicationWindow, version string) {
	d := gtk.NewAboutDialog()
	d.SetTransientFor(&parent.Window)
	d.SetProgramName("WhatsApp (unofficial)")
	d.SetVersion(version)
	d.SetComments("Unofficial native Linux WhatsApp client. Not affiliated with WhatsApp LLC/Meta.")
	d.SetWebsite("https://github.com/Bongbetic/whatsapp-linux")
	d.Show()
}
