// Package theme installs the Windows-desktop look (§9): the embedded
// stylesheet (reference design tokens, light and dark via
// prefers-color-scheme) and the app's own wl-* symbolic icons.
// Embedding keeps dev runs from any directory styled the same as the
// installed package.
package theme

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/rs/zerolog"
)

//go:embed style.css
var css string

//go:embed icons/*.svg
var icons embed.FS

// Install applies the stylesheet to display and makes the wl-* icon
// names resolvable. GTK's icon theme only searches directories, so
// the icons are materialized under cacheDir/icons first.
func Install(display *gdk.Display, cacheDir string, log zerolog.Logger) {
	prov := gtk.NewCSSProvider()
	prov.ConnectParsingError(func(sec *gtk.CSSSection, err error) {
		log.Warn().Err(err).Str("at", sec.String()).Msg("stylesheet")
	})
	if !followScheme(display, prov) {
		prov.LoadFromString(css)
	}
	gtk.StyleContextAddProviderForDisplay(display, prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)

	dir := filepath.Join(cacheDir, "icons")
	if err := writeIcons(dir); err != nil {
		log.Warn().Err(err).Msg("icons")
		return
	}
	gtk.IconThemeGetForDisplay(display).AddSearchPath(dir)
}

// writeIcons copies the embedded SVGs into dir, skipping files that
// are already current.
func writeIcons(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(icons, "icons", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := icons.ReadFile(path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, d.Name())
		if old, err := os.ReadFile(dst); err == nil && bytes.Equal(old, data) {
			return nil
		}
		return os.WriteFile(dst, data, 0o644)
	})
}
