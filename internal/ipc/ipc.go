// Package ipc provides a control channel for inter-process communication.
// On Linux: Unix domain sockets. On Windows: Named pipes.
// This allows a second `clipd` invocation to drive the already-running instance —
// e.g. `clipd toggle` to pop the window open.
package ipc

// Commands understood by the listener.
const (
	CmdToggle = "toggle"
	CmdShow   = "show"
	CmdHide   = "hide"
	CmdQuit   = "quit"
)

// ValidCommand reports whether cmd is a recognised control command.
func ValidCommand(cmd string) bool {
	switch cmd {
	case CmdToggle, CmdShow, CmdHide, CmdQuit:
		return true
	default:
		return false
	}
}

// IsRunning and Send are implemented in ipc_linux.go and ipc_windows.go
// Listener is also implemented in platform-specific files.
