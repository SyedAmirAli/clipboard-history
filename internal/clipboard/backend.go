package clipboard

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Server identifies the display-server protocol clipd is running under.
// On Windows, this is ServerWindows. On Linux, X11 or Wayland.
type Server int

const (
	ServerX11 Server = iota
	ServerWayland
	ServerWindows
)

func (s Server) String() string {
	switch s {
	case ServerWayland:
		return "wayland"
	case ServerWindows:
		return "windows"
	default:
		return "x11"
	}
}

// DetectServer inspects the environment to decide which platform we're on.
// On Windows, returns ServerWindows. On Linux, detects Wayland vs X11.
func DetectServer() Server {
	if runtime.GOOS == "windows" {
		return ServerWindows
	}
	// Linux: detect X11 vs Wayland
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
// set the clipboard for the given server. For Windows, returns empty (built-in).
func requiredTools(s Server) []string {
	if s == ServerWindows {
		return []string{} // Windows uses built-in Win32 API
	}
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
