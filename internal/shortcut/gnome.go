// Package shortcut installs a desktop-level global keyboard shortcut that
// runs `clipd toggle`. It exists because Wayland — unlike X11 — does not
// let an application grab a global hotkey for itself; the binding has to be
// registered with the compositor/desktop instead.
//
// Only GNOME is automated here (via gsettings), since that's the default on
// Ubuntu/Wayland where the in-process X11 grab fails. On other desktops the
// caller falls back to telling the user to bind `clipd toggle` manually.
package shortcut

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Manager adapts the desktop-shortcut mechanism to the service's
// HotkeySetter interface. Under Wayland a client cannot grab a global key
// for itself, so "registering" a hotkey means installing a desktop binding
// (GNOME via gsettings) that runs `clipd toggle`. Replacing the old in-process
// X11 grab, Manager is the single hotkey path now that clipd is Wayland-only.
type Manager struct{}

// NewManager returns a shortcut Manager.
func NewManager() *Manager { return &Manager{} }

// Register installs (or updates) the desktop binding for spec. On desktops we
// can't automate (non-GNOME), it logs guidance and returns nil rather than
// failing — the user binds `clipd toggle` themselves, and a settings save
// shouldn't error just because we can't write the binding for them.
func (m *Manager) Register(spec string) error {
	if !Available() {
		log.Printf("shortcut: automatic install unavailable on this desktop; bind %q to \"clipd toggle\" manually", spec)
		return nil
	}
	return EnsureInstalled(spec)
}

// keybindingPath is the dedicated GSettings relocatable path clipd owns. We
// namespace it under our own name so we never clobber a user's existing
// custom-keybindings (custom0, custom1, …).
const keybindingPath = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/clipd/"

const (
	mediaKeysSchema   = "org.gnome.settings-daemon.plugins.media-keys"
	keybindingSchema  = "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding"
	customKeybindings = "custom-keybindings"
)

// Available reports whether automated shortcut installation is supported on
// this desktop — i.e. we're on GNOME and the gsettings tool is present.
func Available() bool {
	if _, err := exec.LookPath("gsettings"); err != nil {
		return false
	}
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	return strings.Contains(desktop, "gnome") || strings.Contains(desktop, "unity")
}

// EnsureInstalled registers (or updates) the GNOME custom keybinding so that
// pressing `spec` (e.g. "Super+V") runs `clipd toggle`. It is idempotent.
//
// On non-GNOME desktops it returns an error describing the manual
// alternative, which the caller can surface as guidance rather than a fault.
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
	if err := gsettingsSet(rel, "name", "clipd"); err != nil {
		return err
	}
	// Bind bare `clipd` (not `clipd toggle`): the first press launches the GUI
	// and shows it; later presses toggle the already-running window. `clipd
	// toggle` would fail on the first press because, in the daemon model, only
	// the headless daemon is running at login — the GUI hasn't started yet.
	if err := gsettingsSet(rel, "command", clipdCommand()); err != nil {
		return err
	}
	if err := gsettingsSet(rel, "binding", accel); err != nil {
		return err
	}
	return nil
}

// Remove deletes the clipd keybinding from GNOME. Safe to call when nothing
// was installed.
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
	// Best-effort: clear the per-binding values we set.
	rel := keybindingSchema + ":" + keybindingPath
	_ = exec.Command("gsettings", "reset-recursively", rel).Run()
	return nil
}

// clipdCommand returns the command to invoke clipd, preferring an absolute
// path so the binding works regardless of the session's PATH.
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

// parsePathList extracts the quoted object paths from a GVariant array
// literal such as "['/org/.../custom0/', '/org/.../clipd/']" or "@as []".
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

// formatPathList renders paths back into the GVariant array literal that
// gsettings expects, e.g. ['/a/', '/b/'] (or @as [] when empty).
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

// toGtkAccelerator converts clipd's "Super+V" style spec into the GTK
// accelerator syntax gsettings wants, e.g. "<Super>v". Modifier names are
// case-insensitive; the final token is the key.
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
	// GTK accelerators use lowercase letters for plain keys.
	b.WriteString(strings.ToLower(key))
	return b.String(), nil
}
