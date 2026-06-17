//go:build linux

package shortcut

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// keybindingPath is the dedicated GSettings relocatable path clipd owns.
const keybindingPath = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/clipd/"

const (
	mediaKeysSchema   = "org.gnome.settings-daemon.plugins.media-keys"
	keybindingSchema  = "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding"
	customKeybindings = "custom-keybindings"
)

// Available reports whether automated shortcut installation is supported.
func Available() bool {
	if _, err := exec.LookPath("gsettings"); err != nil {
		return false
	}
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	return strings.Contains(desktop, "gnome") || strings.Contains(desktop, "unity")
}

// EnsureInstalled registers the GNOME custom keybinding for the given spec.
func EnsureInstalled(spec string) error {
	if !Available() {
		return fmt.Errorf("automatic shortcut install needs GNOME + gsettings; bind \"%s toggle\" to a key manually in your desktop settings", clipdCommand())
	}
	accel, err := toGtkAccelerator(spec)
	if err != nil {
		return err
	}

	list, err := currentKeybindings()
	if err != nil {
		return err
	}
	if !contains(list, keybindingPath) {
		list = append(list, keybindingPath)
		if err := setKeybindings(list); err != nil {
			return err
		}
	}

	rel := keybindingSchema + ":" + keybindingPath
	if err := gsettingsSet(rel, "name", "clipd toggle"); err != nil {
		return err
	}
	if err := gsettingsSet(rel, "command", clipdCommand()+" toggle"); err != nil {
		return err
	}
	if err := gsettingsSet(rel, "binding", accel); err != nil {
		return err
	}
	return nil
}

// Remove deletes the clipd keybinding from GNOME.
func Remove() error {
	if !Available() {
		return nil
	}
	list, err := currentKeybindings()
	if err != nil {
		return err
	}
	if !contains(list, keybindingPath) {
		return nil
	}
	var kept []string
	for _, p := range list {
		if p != keybindingPath {
			kept = append(kept, p)
		}
	}
	if err := setKeybindings(kept); err != nil {
		return err
	}
	rel := keybindingSchema + ":" + keybindingPath
	_ = exec.Command("gsettings", "reset-recursively", rel).Run()
	return nil
}

// clipdCommand returns the command to invoke clipd.
func clipdCommand() string {
	if p, err := exec.LookPath("clipd"); err == nil {
		return p
	}
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "clipd"
}

func currentKeybindings() ([]string, error) {
	out, err := exec.Command("gsettings", "get", mediaKeysSchema, customKeybindings).Output()
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", customKeybindings, err)
	}
	return parsePathList(string(out)), nil
}

func setKeybindings(paths []string) error {
	return exec.Command("gsettings", "set", mediaKeysSchema, customKeybindings, formatPathList(paths)).Run()
}

func gsettingsSet(schemaWithPath, key, value string) error {
	if err := exec.Command("gsettings", "set", schemaWithPath, key, value).Run(); err != nil {
		return fmt.Errorf("set %s %s: %w", schemaWithPath, key, err)
	}
	return nil
}

func parsePathList(s string) []string {
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

func formatPathList(paths []string) string {
	if len(paths) == 0 {
		return "@as []"
	}
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = "'" + p + "'"
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func toGtkAccelerator(spec string) (string, error) {
	parts := strings.Split(spec, "+")
	if len(parts) < 2 {
		return "", fmt.Errorf("shortcut %q: expected at least one modifier + a key", spec)
	}
	var b strings.Builder
	for _, p := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "super", "meta", "mod4", "win":
			b.WriteString("<Super>")
		case "ctrl", "control":
			b.WriteString("<Control>")
		case "alt", "mod1":
			b.WriteString("<Alt>")
		case "shift":
			b.WriteString("<Shift>")
		default:
			return "", fmt.Errorf("shortcut %q: unknown modifier %q", spec, p)
		}
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	if key == "" {
		return "", fmt.Errorf("shortcut %q: missing key", spec)
	}
	b.WriteString(strings.ToLower(key))
	return b.String(), nil
}
