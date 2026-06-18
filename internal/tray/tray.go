// Package tray installs a system tray (AppIndicator) icon with a menu
// that mirrors the Wails app's primary actions.
//
// On Linux, fyne.io/systray uses libayatana-appindicator under the hood.
// Wails already owns the process-wide GTK main loop, so we call
// systray.Register (NOT Run) — Register sets up the indicator without
// starting its own GTK loop, and Wails's running loop pumps the events.
package tray

import (
	_ "embed"

	"fyne.io/systray"
)

// Callbacks bundles the click handlers wired up on tray menu items.
type Callbacks struct {
	OnOpen     func()
	OnClear    func()
	OnSettings func()
	OnQuit     func()
}

// Windows' system tray requires ICO-format icon bytes; a PNG fails to load
// ("unable to set icon"), leaving a blank/default icon.
//
//go:embed icon.ico
var iconICO []byte

// Start installs the tray indicator and wires its menu items to cb.
// It returns immediately — menu click pumping happens on a goroutine.
//
// Start MUST be called only after Wails has initialised GTK (i.e. from
// the OnStartup hook), otherwise the AppIndicator handle would have
// nothing to attach to.
func Start(cb Callbacks) {
	systray.Register(func() {
		systray.SetIcon(iconICO)
		systray.SetTitle("clipd")
		systray.SetTooltip("Clipboard History")

		mOpen := systray.AddMenuItem("Open clipboard history", "Show the history popup")
		mClear := systray.AddMenuItem("Clear history", "Delete all non-pinned items")
		mSettings := systray.AddMenuItem("Settings…", "Configure clipd")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit clipd", "Exit the application")

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					if cb.OnOpen != nil {
						cb.OnOpen()
					}
				case <-mClear.ClickedCh:
					if cb.OnClear != nil {
						cb.OnClear()
					}
				case <-mSettings.ClickedCh:
					if cb.OnSettings != nil {
						cb.OnSettings()
					}
				case <-mQuit.ClickedCh:
					if cb.OnQuit != nil {
						cb.OnQuit()
					}
					return
				}
			}
		}()
	}, func() {
		// onExit: no-op; Wails handles process teardown.
	})
}

// Quit removes the tray indicator. Call before the app exits if you
// want the icon to disappear immediately.
func Quit() {
	systray.Quit()
}
