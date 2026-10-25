package settings

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/media"
)

// StorageSection renders the WL-3.3 storage page fragment: usage by
// chat plus clear-cache. Appended to the settings stack by the shell.
func StorageSection(dataDir, cacheDir string) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 6)
	box.Append(gtk.NewLabel("Storage"))
	usage := gtk.NewLabel("calculating…")
	box.Append(usage)
	refresh := func() {
		m, err := media.UsageByChat(dataDir)
		if err != nil {
			usage.SetText("usage unavailable: " + err.Error())
			return
		}
		var total int64
		for _, b := range m {
			total += b
		}
		usage.SetText(fmt.Sprintf("%d chats use %s of media", len(m), humanBytes(total)))
	}
	refresh()
	clear := gtk.NewButtonWithLabel("Clear thumbnail cache")
	clear.ConnectClicked(func() {
		if err := media.ClearCache(cacheDir); err != nil {
			usage.SetText("clear failed: " + err.Error())
			return
		}
		refresh()
	})
	box.Append(clear)
	return box
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
