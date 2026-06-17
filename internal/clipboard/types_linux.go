//go:build linux

package clipboard

// DefaultSettings returns Linux-specific default settings.
func DefaultSettings() Settings {
	return Settings{
		Hotkey:      "Super+V",  // Super+V is the traditional hotkey on Linux
		MaxItems:    200,
		KeepImages:  true,
		MaxImageMB:  5,
		Theme:       "auto",
		Autostart:   false,
		HideOnBlur:  true,
		LaunchAtTop: false,
		WindowFrame: true,
		AutoPaste:   true,
	}
}
