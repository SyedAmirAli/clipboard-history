package clipboard

import (
	"os"
	"os/exec"
	"strings"
)

// Server identifies the display-server protocol clipd is running under.
// The external clipboard tools differ between the two: X11 uses
// xclip/xdotool, Wayland uses wl-clipboard (wl-paste / wl-copy). clipd
// ships as a single binary and picks the right backend at runtime — the
// display server is a session-level choice (you can run X11 or Wayland on
// the same distro), so it must be detected, never baked in at build time.
type Server int

const (
	ServerX11 Server = iota
	ServerWayland
)

func (s Server) String() string {
	if s == ServerWayland {
		return "wayland"
	}
	return "x11"
}

// DetectServer inspects the environment to decide whether we're on Wayland
// or X11. WAYLAND_DISPLAY is the most reliable signal — it's exported by the
// compositor for every client — with XDG_SESSION_TYPE as a fallback for odd
// setups. Anything else is treated as X11, the safe default for legacy
// desktops (and the only one xclip/xdotool can drive).
func DetectServer() Server {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return ServerWayland
	}
	if strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") {
		return ServerWayland
	}
	return ServerX11
}

// IsWayland reports whether the current session is Wayland.
func IsWayland() bool { return DetectServer() == ServerWayland }

// requiredTools returns the external commands clipd needs to capture and
// set the clipboard for the given server. Missing ones make history
// capture impossible, so the caller surfaces them to the user.
func requiredTools(s Server) []string {
	if s == ServerWayland {
		return []string{"wl-paste", "wl-copy"}
	}
	return []string{"xclip"}
}

// MissingTools returns any of the server's required commands that are not
// found in PATH. An empty slice means the backend is ready to use.
func MissingTools(s Server) []string {
	var missing []string
	for _, t := range requiredTools(s) {
		if _, err := exec.LookPath(t); err != nil {
			missing = append(missing, t)
		}
	}
	return missing
}
