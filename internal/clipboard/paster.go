package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var pasteTarget struct {
	sync.Mutex
	window string
}

// RememberPasteTarget records the active X11/XWayland window before clipd takes
// focus. On native Wayland applications, xdotool cannot see or control the
// focused surface, so this remains a best-effort feature.
func RememberPasteTarget() {
	if os.Getenv("DISPLAY") == "" {
		return
	}
	out, err := exec.Command("xdotool", "getactivewindow").Output()
	if err != nil {
		return
	}
	id := strings.TrimSpace(string(out))
	if id == "" || isClipdWindow(id) {
		return
	}
	pasteTarget.Lock()
	pasteTarget.window = id
	pasteTarget.Unlock()
}

// SendPaste activates the remembered window and sends Ctrl+V after the selected
// item has been written to the clipboard. This works for X11/XWayland clients.
// GNOME Wayland intentionally blocks arbitrary input injection into native
// Wayland windows; supporting those would require a privileged virtual-input
// daemon such as ydotool, which is outside clipd's normal app boundary.
func SendPaste() error {
	if os.Getenv("DISPLAY") == "" {
		return nil
	}
	if _, err := exec.LookPath("xdotool"); err != nil {
		return fmt.Errorf("xdotool not found; install xdotool or turn off Auto-paste")
	}
	target := rememberedPasteTarget()
	if target != "" {
		_ = exec.Command("xdotool", "windowactivate", "--sync", target).Run()
		time.Sleep(80 * time.Millisecond)
	}
	if err := exec.Command("xdotool", "key", "--clearmodifiers", "ctrl+v").Run(); err != nil {
		return fmt.Errorf("send Ctrl+V: %w", err)
	}
	return nil
}

func rememberedPasteTarget() string {
	pasteTarget.Lock()
	defer pasteTarget.Unlock()
	return pasteTarget.window
}

func isClipdWindow(id string) bool {
	out, err := exec.Command("xdotool", "getwindowclassname", id).Output()
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(out)), "clipd")
}
