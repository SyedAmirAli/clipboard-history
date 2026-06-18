// Package tray installs a Windows system-tray icon with a menu that mirrors the
// app's primary actions.
//
// We run systray on its own dedicated OS thread via systray.Run (NOT Register).
// On Windows the tray needs its own GetMessage loop to dispatch icon clicks and
// the right-click menu; Register relies on the host (Wails) loop, which does not
// pump the tray window's messages, so the menu never appears. Running on a
// locked thread keeps the tray loop independent of Wails' main loop.
package tray

import (
	_ "embed"
	"runtime"

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

// Start installs the tray icon and wires its menu items to cb. It returns
// immediately; the tray's own message loop runs on a dedicated, OS-locked
// goroutine so it functions independently of Wails' main loop.
func Start(cb Callbacks) {
	go func() {
		// The tray window and its message loop must live on the same OS thread.
		runtime.LockOSThread()
		systray.Run(func() {
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
	}()
}

// Quit removes the tray icon. Call before the app exits if you want the icon to
// disappear immediately.
func Quit() {
	systray.Quit()
}
