package clipboard

// SendPaste synthesises a Ctrl+V key event into the focused window.
// Platform-specific implementations in paster_linux.go and paster_windows.go.
func SendPaste() error {
	panic("SendPaste must be implemented by platform-specific code")
}
