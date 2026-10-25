// Package autostart writes the XDG autostart entry (WL-2.5).
package autostart

import (
	"os"
	"path/filepath"
)

// DesktopFileID is the autostart entry name.
const DesktopFileID = "com.bongbetic.WhatsAppLinux.desktop"

// Dir returns the XDG autostart dir.
func Dir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "autostart")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "autostart")
}

// Set creates (enabled) or removes (disabled) the autostart entry.
// The entry launches hidden to the tray.
func Set(enabled bool) error {
	path := filepath.Join(Dir(), DesktopFileID)
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	body := "[Desktop Entry]\nType=Application\nName=WhatsApp (unofficial)\nExec=" + exe + " --hidden\nIcon=com.bongbetic.WhatsAppLinux\nX-GNOME-Autostart-enabled=true\n"
	return os.WriteFile(path, []byte(body), 0o644)
}
