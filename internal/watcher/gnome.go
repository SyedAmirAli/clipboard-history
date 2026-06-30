// Package watcher contains small integration hooks for the external clipboard
// watcher. The watcher itself is a GNOME Shell extension plus a Python DB
// helper; this package only enables the already-installed extension.
package watcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const ExtensionUUID = "clipd-watcher@syedamirali"

// EnsureEnabled enables clipd's GNOME Shell extension for the current user.
// It is intentionally best-effort: systems without GNOME, without the
// gnome-extensions CLI, or without the packaged extension should still be able
// to run the GUI and use manually configured capture.
func EnsureEnabled() error {
	if !isGNOMESession() {
		return nil
	}
	if _, err := exec.LookPath("gnome-extensions"); err != nil {
		return nil
	}
	if _, err := os.Stat("/usr/share/gnome-shell/extensions/" + ExtensionUUID); err != nil {
		return nil
	}
	if exec.Command("gnome-extensions", "enable", ExtensionUUID).Run() == nil {
		return nil
	}
	// Older or restricted sessions may not report system extensions via the CLI.
	// Fall back to directly adding the UUID to org.gnome.shell enabled-extensions.
	out, err := exec.Command("gsettings", "get", "org.gnome.shell", "enabled-extensions").Output()
	if err != nil {
		return fmt.Errorf("enable GNOME extension %s: %w", ExtensionUUID, err)
	}
	list := parseGSettingsStringArray(string(out))
	if contains(list, ExtensionUUID) {
		return nil
	}
	list = append(list, ExtensionUUID)
	if err := exec.Command("gsettings", "set", "org.gnome.shell", "enabled-extensions", formatGSettingsStringArray(list)).Run(); err != nil {
		return fmt.Errorf("enable GNOME extension %s: %w", ExtensionUUID, err)
	}
	return nil
}

func isGNOMESession() bool {
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	return strings.Contains(desktop, "gnome") || strings.Contains(desktop, "unity")
}

func parseGSettingsStringArray(s string) []string {
	var out []string
	for {
		i := strings.IndexByte(s, '\'')
		if i < 0 {
			break
		}
		s = s[i+1:]
		j := strings.IndexByte(s, '\'')
		if j < 0 {
			break
		}
		out = append(out, s[:j])
		s = s[j+1:]
	}
	return out
}

func formatGSettingsStringArray(values []string) string {
	if len(values) == 0 {
		return "@as []"
	}
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "'" + strings.ReplaceAll(v, "'", "\\'") + "'"
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
