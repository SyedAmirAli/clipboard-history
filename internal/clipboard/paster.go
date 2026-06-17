package clipboard

// SaveFocusedWindow stores the currently focused window before showing the popup.
// On Windows, this allows us to restore focus before SendPaste so the paste
// goes to the correct application. On Linux, this is a no-op.
// Platform-specific implementations in paster_linux.go and paster_windows.go.

// SendPaste synthesises a Ctrl+V key event into the focused window.
// Platform-specific implementations in paster_linux.go and paster_windows.go.
