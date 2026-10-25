// Package business renders the business profile dialog (WL-4.5):
// address, hours, categories, description. Catalog/product messages
// and cart send stay scoped to eligibility (Phase 8).
package business

import (
	"context"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// Show opens the business profile dialog for a contact.
func Show(parent *gtk.ApplicationWindow, eng bridge.Engine, biz bridge.JID) {
	d := gtk.NewDialog()
	d.SetTransientFor(&parent.Window)
	d.SetTitle("Business profile")
	box := gtk.NewBox(gtk.OrientationVertical, 6)
	status := gtk.NewLabel("loading…")
	box.Append(status)
	d.SetChild(box)
	d.Show()
	go func() {
		p, err := eng.BusinessProfile(context.Background(), biz)
		text := "error: " + errString(err)
		if err == nil {
			lines := []string{p.Description, p.Address, p.Email}
			if len(p.Categories) > 0 {
				lines = append(lines, "Categories: "+strings.Join(p.Categories, ", "))
			}
			text = strings.Join(nonEmpty(lines), "\n")
		}
		glib.IdleAdd(func() bool {
			status.SetText(text)
			return false
		})
	}()
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func nonEmpty(in []string) []string {
	var out []string
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return []string{"(no public profile)"}
	}
	return out
}
