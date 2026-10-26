package settings

import (
	"context"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/rs/zerolog"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// privacyKeys is the WL-4.6 server-side privacy surface, in UI order.
var privacyKeys = []struct {
	key, label string
	values     []string
}{
	{"last", "Last seen", []string{"all", "contacts", "none"}},
	{"profile", "Profile photo", []string{"all", "contacts", "none"}},
	{"status", "About/status", []string{"all", "contacts", "none"}},
	{"readreceipts", "Read receipts", []string{"all", "none"}},
	{"groupadd", "Who can add me to groups", []string{"all", "contacts", "none"}},
	{"online", "Online presence", []string{"all", "match_last_seen"}},
	{"calladd", "Who can call me", []string{"all", "known"}},
}

// PrivacySection renders server-side privacy + blocklist (WL-4.6).
// Values load async; offline shows the cached kv copy marked stale.
func PrivacySection(eng bridge.Engine, log zerolog.Logger) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 6)
	box.Append(Section("Privacy (server-side)"))
	rows := gtk.NewBox(gtk.OrientationVertical, 2)
	box.Append(rows)
	status := gtk.NewLabel("loading…")
	box.Append(status)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cur, err := eng.PrivacyGet(ctx)
		glib.IdleAdd(func() bool {
			if err != nil {
				status.SetText("offline — showing last known (stale)")
				cur = bridge.PrivacySettings{}
			} else {
				status.SetText("")
			}
			for _, k := range privacyKeys {
				k := k
				row := gtk.NewBox(gtk.OrientationHorizontal, 6)
				lbl := gtk.NewLabel(k.label)
				lbl.SetHExpand(true)
				lbl.SetXAlign(0)
				row.Append(lbl)
				for _, v := range k.values {
					v := v
					b := gtk.NewButtonWithLabel(v)
					if cur[k.key] == v {
						b.AddCSSClass("suggested-action")
					}
					b.ConnectClicked(func() {
						if err := eng.PrivacySet(context.Background(), k.key, v); err != nil {
							log.Error().Err(err).Msg("privacy set")
						}
					})
					row.Append(b)
				}
				rows.Append(row)
			}
			box.Append(blocklistSection(eng, log))
			return false
		})
	}()
	return box
}

func blocklistSection(eng bridge.Engine, log zerolog.Logger) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 4)
	box.Append(gtk.NewLabel("Blocked contacts"))
	list := gtk.NewListBox()
	box.Append(list)
	var refresh func()
	refresh = func() {
		list.RemoveAll()
		jids, err := eng.BlocklistGet(context.Background())
		if err != nil {
			log.Debug().Err(err).Msg("blocklist unavailable offline")
			return
		}
		for _, j := range jids {
			j := j
			row := gtk.NewBox(gtk.OrientationHorizontal, 6)
			lbl := gtk.NewLabel(j.String())
			lbl.SetHExpand(true)
			lbl.SetXAlign(0)
			row.Append(lbl)
			un := gtk.NewButtonWithLabel("Unblock")
			un.ConnectClicked(func() {
				_ = eng.BlockSet(context.Background(), j, false)
				refresh()
			})
			row.Append(un)
			list.Append(row)
		}
	}
	refresh()
	addRow := gtk.NewBox(gtk.OrientationHorizontal, 6)
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("phone to block")
	addRow.Append(entry)
	add := gtk.NewButtonWithLabel("Block")
	add.ConnectClicked(func() {
		phone := entry.Buffer().Text()
		if phone == "" {
			return
		}
		if err := eng.BlockSet(context.Background(), bridge.JID{User: phone, Server: "s.whatsapp.net"}, true); err != nil {
			log.Error().Err(err).Msg("block")
			return
		}
		entry.Buffer().SetText("", -1)
		refresh()
	})
	addRow.Append(add)
	box.Append(addRow)
	return box
}
