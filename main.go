package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"clipd/internal/autostart"
	"clipd/internal/clipboard"
	"clipd/internal/config"
	"clipd/internal/db"
	"clipd/internal/hotkey"
	"clipd/internal/service"
	"clipd/internal/tray"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	dbPath, err := config.DBPath()
	if err != nil {
		log.Fatalf("resolve db path: %v", err)
	}
	store, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	watcher := clipboard.NewWatcher(500_000_000) // 500 ms
	hk := hotkey.New()
	as := autostart.New()

	svc := service.New(store, hk, watcher, as)

	app := &appWiring{
		svc:     svc,
		watcher: watcher,
		hotkey:  hk,
	}

	err = wails.Run(&options.App{
		Title:             "clipd",
		Width:             520,
		Height:            620,
		MinWidth:          380,
		MinHeight:         420,
		DisableResize:     false,
		Frameless:         true,
		StartHidden:       true,
		HideWindowOnClose: true,
		AlwaysOnTop:       true,
		BackgroundColour:  &options.RGBA{R: 22, G: 23, B: 30, A: 0},
		AssetServer:       &assetserver.Options{Assets: assets},
		OnStartup:         app.onStartup,
		OnShutdown:        app.onShutdown,
		OnBeforeClose:     app.onBeforeClose,
		Bind:              []any{svc},
		Linux: &linux.Options{
			Icon:                appIcon,
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyAlways,
			ProgramName:         "clipd",
		},
	})
	if err != nil {
		log.Fatalf("wails run: %v", err)
	}
}

// appWiring holds long-lived components so the OnStartup/OnShutdown
// hooks can finish wiring them up against the Wails context.
type appWiring struct {
	svc     *service.Service
	watcher *clipboard.Watcher
	hotkey  *hotkey.Manager
}

func (a *appWiring) onStartup(ctx context.Context) {
	a.svc.AttachContext(ctx)

	// Hard-fail-soft on Wayland: clipboard monitoring and X11 hotkeys
	// won't work; surface a friendly message and continue in tray-only
	// mode (the user can still open the window via the tray icon).
	sessType := os.Getenv("XDG_SESSION_TYPE")
	wayland := sessType == "wayland"

	if !wayland {
		settings := a.svc.CurrentSettings()
		if err := a.hotkey.Register(settings.Hotkey); err != nil {
			log.Printf("hotkey register: %v", err)
			wailsruntime.LogWarning(ctx, "Failed to register hotkey: "+err.Error())
		}
		if err := a.watcher.Start(ctx); err != nil {
			log.Printf("watcher start: %v", err)
			wailsruntime.LogWarning(ctx, "Failed to start clipboard watcher: "+err.Error())
		}
		go a.consumeClipboard(ctx)
		go a.consumeHotkey(ctx)
	} else {
		wailsruntime.LogWarning(ctx, "Running on Wayland — global hotkey and background clipboard monitoring are disabled. Use the tray icon to open the popup.")
	}

	tray.Start(tray.Callbacks{
		OnOpen:     a.svc.ShowPopup,
		OnClear:    func() { _ = a.svc.ClearAll(true) },
		OnSettings: a.svc.ShowPopup,
		OnQuit:     func() { wailsruntime.Quit(ctx) },
	})
}

func (a *appWiring) onShutdown(ctx context.Context) {
	tray.Quit()
	a.hotkey.Close()
	a.watcher.Stop()
}

func (a *appWiring) onBeforeClose(ctx context.Context) bool {
	// Returning true keeps the window open. We always hide instead so
	// the app continues to live in the tray.
	a.svc.HidePopup()
	return true
}

func (a *appWiring) consumeClipboard(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-a.watcher.Events():
			if !ok {
				return
			}
			var err error
			switch ev.ContentType {
			case clipboard.ContentTypeText:
				err = a.svc.IngestText(ev.Text, ev.Hash)
			case clipboard.ContentTypeImage:
				err = a.svc.IngestImage(ev.ImagePNG, ev.Hash)
			}
			if err != nil {
				log.Printf("ingest %s: %v", ev.ContentType, err)
			}
		}
	}
}

func (a *appWiring) consumeHotkey(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.hotkey.Events():
			a.svc.TogglePopup()
		}
	}
}
