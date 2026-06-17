//go:build windows

package clipboard

// DefaultSettings returns Windows-specific default settings.
func DefaultSettings() Settings {
	return Settings{
		Hotkey:      "Win+J",  // Win+J is commonly free on Windows
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
