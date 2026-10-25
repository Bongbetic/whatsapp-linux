// Package groups renders group management (WL-4.1): member list with
// roles, add/remove/promote/demote, invite link display + reset,
// rename/topic edit, and the create-group dialog.
package groups

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// Manage opens the group management dialog for group.
func Manage(parent *gtk.ApplicationWindow, eng bridge.Engine, group bridge.JID) {
	ctx := context.Background()
	meta, err := eng.GroupMeta(ctx, group)
	d := gtk.NewDialog()
	d.SetTransientFor(&parent.Window)
	d.SetTitle("Group")
	d.SetDefaultSize(420, 480)
	box := gtk.NewBox(gtk.OrientationVertical, 6)

	name := gtk.NewEntry()
	if err == nil {
		name.SetText(meta.Name)
	} else {
		name.SetText("(load failed: " + err.Error() + ")")
	}
	box.Append(name)
	save := gtk.NewButtonWithLabel("Rename")
	save.ConnectClicked(func() {
		_ = eng.GroupSetName(ctx, group, name.Buffer().Text())
	})
	box.Append(save)

	list := gtk.NewListBox()
	var fill func()
	fill = func() {
		list.RemoveAll()
		m, err := eng.GroupMeta(ctx, group)
		if err != nil {
			return
		}
		for _, mem := range m.Members {
			mem := mem
			row := gtk.NewBox(gtk.OrientationHorizontal, 6)
			lbl := gtk.NewLabel(mem.User.String() + " — " + mem.Role)
			lbl.SetHExpand(true)
			lbl.SetXAlign(0)
			row.Append(lbl)
			for _, act := range []struct {
				label, op string
			}{{"+", "promote"}, {"−", "demote"}, {"✕", "remove"}} {
				act := act
				b := gtk.NewButtonWithLabel(act.label)
				b.ConnectClicked(func() {
					_ = eng.GroupMembers(ctx, group, []bridge.JID{mem.User}, act.op)
					fill()
				})
				row.Append(b)
			}
			list.Append(row)
		}
	}
	fill()
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetChild(list)
	box.Append(scroll)

	addRow := gtk.NewBox(gtk.OrientationHorizontal, 6)
	addEntry := gtk.NewEntry()
	addEntry.SetPlaceholderText("phone number to add")
	addRow.Append(addEntry)
	addBtn := gtk.NewButtonWithLabel("Add")
	addBtn.ConnectClicked(func() {
		phone := addEntry.Buffer().Text()
		_ = eng.GroupMembers(ctx, group, []bridge.JID{{User: phone, Server: "s.whatsapp.net"}}, "add")
		fill()
	})
	addRow.Append(addBtn)
	box.Append(addRow)

	linkLbl := gtk.NewLabel("")
	box.Append(linkLbl)
	linkBtn := gtk.NewButtonWithLabel("Show invite link")
	linkBtn.ConnectClicked(func() {
		link, err := eng.GroupInviteLink(ctx, group, false)
		if err != nil {
			linkLbl.SetText("error: " + err.Error())
			return
		}
		linkLbl.SetText(link)
	})
	box.Append(linkBtn)

	d.SetChild(box)
	d.Show()
}

// Create opens the new-group dialog (member phones comma-separated).
func Create(parent *gtk.ApplicationWindow, eng bridge.Engine, onDone func(bridge.JID)) {
	d := gtk.NewDialog()
	d.SetTransientFor(&parent.Window)
	d.SetTitle("New group")
	box := gtk.NewBox(gtk.OrientationVertical, 6)
	name := gtk.NewEntry()
	name.SetPlaceholderText("Group name (≤25 chars)")
	box.Append(name)
	members := gtk.NewEntry()
	members.SetPlaceholderText("member phones, comma separated")
	box.Append(members)
	go_ := gtk.NewButtonWithLabel("Create")
	go_.ConnectClicked(func() {
		var ms []bridge.JID
		for _, p := range splitPhones(members.Buffer().Text()) {
			ms = append(ms, bridge.JID{User: p, Server: "s.whatsapp.net"})
		}
		jid, err := eng.CreateGroup(context.Background(), name.Buffer().Text(), ms)
		if err != nil {
			return
		}
		d.Close()
		onDone(jid)
	})
	box.Append(go_)
	d.SetChild(box)
	d.Show()
}

func splitPhones(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' || r == ' ' || r == ';' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		if r >= '0' && r <= '9' || r == '+' {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
