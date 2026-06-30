package clipboard

import "os/exec"

// clipd is Wayland-only. It drives the system clipboard exclusively through
// wl-clipboard (wl-paste / wl-copy); the old X11 path (xclip/xdotool) has been
// removed. We deliberately avoid linking libwayland/libX11 via cgo so the
// produced binary stays small — the CLI tools are declared as runtime
// dependencies in the .deb package.

// requiredTools returns the external commands clipd needs to capture and set
// the clipboard. Missing ones make history capture impossible, so the caller
// surfaces them to the user.
func requiredTools() []string {
	return []string{"wl-paste", "wl-copy"}
}

// MissingTools returns any required command that is not found in PATH. An
// empty slice means the backend is ready to use.
func MissingTools() []string {
	var missing []string
	for _, t := range requiredTools() {
		if _, err := exec.LookPath(t); err != nil {
			missing = append(missing, t)
		}
	}
	return missing
}

// installHint returns the apt package that provides the clipboard tools, used
// in user-facing "not installed" messages.
func installHint() string { return "'sudo apt install wl-clipboard'" }
