package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"runtime"

	"clipd/internal/autostart"
	"clipd/internal/clipboard"
	"clipd/internal/config"
	"clipd/internal/db"
	"clipd/internal/hotkey"
	"clipd/internal/ipc"
	"clipd/internal/service"
	"clipd/internal/shortcut"
	"clipd/internal/tray"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	sockPath, _ := config.SocketPath()

	// CLI control mode: `clipd toggle|show|hide` forwards the command to a
	// running instance and exits. This is the alternative to the global
	// Super+V hotkey for environments where X11 hotkey grabs don't work
	// (Wayland, WSLg) or for users who prefer binding a command.
	explicitStart := false
	if len(os.Args) > 1 {
		if sockPath == "" {
			log.Fatal("clipd: cannot resolve control socket path")
		}
		switch arg := os.Args[1]; arg {
		case "start":
			// Explicit start: fall through to daemon startup below. Unlike a
			// bare `clipd`, if one is already running we just report it
			// instead of toggling the window.
			explicitStart = true
		case "toggle", "show", "hide":
			if err := ipc.Send(sockPath, arg); err != nil {
				log.Fatalf("clipd: no running instance to %q (is clipd started?): %v", arg, err)
			}
			return
		case "quit", "exit", "stop":
			if err := ipc.Send(sockPath, ipc.CmdQuit); err != nil {
				log.Fatalf("clipd: no running instance to quit: %v", err)
			}
			log.Println("clipd: shutdown requested")
			return
		case "restart":
			// Ask any running instance to quit, wait for it to release the
			// socket, then start a fresh detached instance.
			_ = ipc.Send(sockPath, ipc.CmdQuit)
			if !waitUntilStopped(sockPath, 5*time.Second) {
				log.Fatal("clipd: previous instance did not shut down in time")
			}
			if err := launchDetached(); err != nil {
				log.Fatalf("clipd: failed to relaunch: %v", err)
			}
			log.Println("clipd: restarted")
			return
		case "reset-vault":
			if ipc.IsRunning(sockPath) {
				log.Fatal("clipd: running instance detected; run 'clipd quit' before reset-vault")
			}
			if err := resetVault(); err != nil {
				log.Fatalf("clipd: reset-vault: %v", err)
			}
			log.Println("clipd: private vault reset")
			return
		case "install-shortcut":
			// Register the desktop-level global shortcut (GNOME/Wayland). The
			// optional second arg overrides the key spec, default Super+V.
			spec := "Super+V"
			if len(os.Args) > 2 {
				spec = os.Args[2]
			}
			if err := shortcut.EnsureInstalled(spec); err != nil {
				log.Fatalf("clipd: install-shortcut: %v", err)
			}
			log.Printf("clipd: installed shortcut %q → clipd toggle", spec)
			return
		case "remove-shortcut":
			if err := shortcut.Remove(); err != nil {
				log.Fatalf("clipd: remove-shortcut: %v", err)
			}
			log.Println("clipd: removed desktop shortcut")
			return
		case "-h", "--help", "help":
			printUsage()
			return
		default:
			fmt.Fprintf(os.Stderr, "clipd: unknown command %q\n\n", arg)
			printUsage()
			os.Exit(2)
		}
	}

	// Single-instance guard: if a clipd is already running, don't start a
	// second copy. `clipd start` reports it; a bare `clipd` toggles the window.
	if sockPath != "" && ipc.IsRunning(sockPath) {
		if explicitStart {
			log.Println("clipd is already running")
		} else if err := ipc.Send(sockPath, ipc.CmdToggle); err == nil {
			log.Println("clipd already running — toggled existing window")
		}
		return
	}

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

	// The window is always frameless — the frontend draws its own title bar
	// (macOS-style traffic lights). The WindowFrame setting selects behaviour:
	// windowed mode = a normal taskbar window; popup mode = an always-on-top
	// Windows+V style overlay that hides on focus loss.
	windowed := svc.CurrentSettings().WindowFrame

	app := &appWiring{
		svc:      svc,
		watcher:  watcher,
		hotkey:   hk,
		sockPath: sockPath,
	}

	opts := &options.App{
		Title:             "clipd",
		Width:             520,
		Height:            620,
		MinWidth:          380,
		MinHeight:         420,
		DisableResize:     false,
		Frameless:         true, // custom titlebar in the frontend; no native chrome
		StartHidden:       true,
		HideWindowOnClose: true,
		AlwaysOnTop:       !windowed,
		BackgroundColour:  &options.RGBA{R: 22, G: 23, B: 30, A: 0},
		AssetServer:       &assetserver.Options{Assets: assets},
		OnStartup:         app.onStartup,
		OnShutdown:        app.onShutdown,
		OnBeforeClose:     app.onBeforeClose,
		Bind:              []any{svc},
	}

	// Platform-specific options
	if runtime.GOOS == "windows" {
		opts.Windows = &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Mica,
		}
	} else {
		opts.Linux = &linux.Options{
			Icon: appIcon,
			WindowIsTranslucent: true,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyAlways,
			ProgramName:         "clipd",
		}
	}

	err = wails.Run(opts)
	if err != nil {
		log.Fatalf("wails run: %v", err)
	}
}

func resetVault() error {
	dbPath, err := config.DBPath()
	if err != nil {
		return fmt.Errorf("resolve db path: %w", err)
	}
	store, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()
	return store.ResetVault()
}

// appWiring holds long-lived components so the OnStartup/OnShutdown
// hooks can finish wiring them up against the Wails context.
type appWiring struct {
	svc         *service.Service
	watcher     *clipboard.Watcher
	hotkey      *hotkey.Manager
	sockPath    string
	ipcLn       *ipc.Listener
	quitting    atomic.Bool
	cleanupOnce sync.Once
}

// cleanup releases all long-lived resources. Safe to call more than once.
func (a *appWiring) cleanup() {
	a.cleanupOnce.Do(func() {
		tray.Quit()
		a.hotkey.Close()
		a.watcher.Stop()
		a.ipcLn.Close()
	})
}

// quit performs a real, full shutdown. It sets the quitting flag so that
// OnBeforeClose lets the window close instead of hiding it, then asks Wails to
// quit. Because a hidden GTK window doesn't always tear down the main loop on
// some setups (notably WSLg), a short grace timer forces the process to exit.
func (a *appWiring) quit(ctx context.Context) {
	a.quitting.Store(true)
	log.Println("clipd: quit requested — shutting down")
	// Arm the forced-exit timer BEFORE calling Quit: on some setups
	// wailsruntime.Quit blocks and never returns, so this must already be
	// running to guarantee the process actually terminates. os.Exit is
	// unconditional here — cleanup is best-effort and must never be able to
	// wedge the shutdown (the OS reclaims fds/sockets on exit anyway).
	go func() {
		time.Sleep(500 * time.Millisecond)
		go a.cleanup() // best-effort, may block; don't let it gate exit
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}()
	wailsruntime.Quit(ctx)
}

func (a *appWiring) onStartup(ctx context.Context) {
	a.svc.AttachContext(ctx)

	// Start the control socket so `clipd toggle|show|hide` can drive this
	// instance — the command-line alternative to the global hotkey.
	if a.sockPath != "" {
		ln, err := ipc.Listen(a.sockPath, func(cmd string) {
			switch cmd {
			case ipc.CmdShow:
				a.svc.ShowPopup()
			case ipc.CmdHide:
				a.svc.HidePopup()
			case ipc.CmdQuit:
				a.quit(ctx)
			default:
				a.svc.TogglePopup()
			}
		})
		if err != nil {
			log.Printf("ipc listen: %v", err)
			wailsruntime.LogWarning(ctx, "Failed to start control socket: "+err.Error())
		} else {
			a.ipcLn = ln
		}
	}

	// Clipboard capture works on both X11 (xclip) and Wayland (wl-clipboard);
	// the watcher picks the backend itself. Surface a friendly message if the
	// required tools aren't installed instead of failing silently.
	server := clipboard.DetectServer()
	if missing := clipboard.MissingTools(server); len(missing) > 0 {
		msg := fmt.Sprintf("Clipboard tools missing for %s session: %s. Install them to capture history.",
			server, strings.Join(missing, ", "))
		log.Print(msg)
		wailsruntime.LogWarning(ctx, msg)
	}
	if err := a.watcher.Start(ctx); err != nil {
		log.Printf("watcher start: %v", err)
		wailsruntime.LogWarning(ctx, "Failed to start clipboard watcher: "+err.Error())
	}
	go a.consumeClipboard(ctx)

	settings := a.svc.CurrentSettings()
	if server == clipboard.ServerWayland {
		// Wayland forbids X11-style global key grabs, so register the shortcut
		// with the desktop (GNOME via gsettings) to run `clipd toggle`. On
		// other desktops, tell the user how to bind it themselves.
		if err := shortcut.EnsureInstalled(settings.Hotkey); err != nil {
			log.Printf("desktop shortcut: %v", err)
			wailsruntime.LogWarning(ctx, "Global hotkey needs a desktop binding on Wayland. "+err.Error())
		}
	} else {
		if err := a.hotkey.Register(settings.Hotkey); err != nil {
			log.Printf("hotkey register: %v", err)
			wailsruntime.LogWarning(ctx, "Failed to register hotkey: "+err.Error())
		}
		go a.consumeHotkey(ctx)
	}

	tray.Start(tray.Callbacks{
		OnOpen:     a.svc.ShowPopup,
		OnClear:    func() { _ = a.svc.ClearAll(true) },
		OnSettings: a.svc.ShowPopup,
		OnQuit:     func() { a.quit(ctx) },
	})
}

func (a *appWiring) onShutdown(ctx context.Context) {
	a.cleanup()
}

// waitUntilStopped polls the control socket until no instance answers or the
// timeout elapses. Returns true once the previous instance is gone.
func waitUntilStopped(sockPath string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !ipc.IsRunning(sockPath) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return !ipc.IsRunning(sockPath)
}

// launchDetached starts a fresh clipd process fully detached from the current
// one (new session), so it keeps running after this CLI invocation exits.
func launchDetached() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func printUsage() {
	fmt.Print(`clipd — clipboard history manager

Usage:
  clipd            Start clipd (or toggle the window if already running)
  clipd start      Start clipd (reports if it's already running)
  clipd toggle     Show/hide the clipboard popup of the running instance
  clipd show       Show the popup
  clipd hide       Hide the popup
  clipd quit       Fully shut down the running instance (aliases: exit, stop)
  clipd restart    Shut down the running instance and start a fresh one
  clipd reset-vault
                   Delete private vault setup and vault entries only
  clipd install-shortcut [spec]
                   Bind a desktop global shortcut (GNOME/Wayland) to
                   "clipd toggle". Defaults to Super+V.
  clipd remove-shortcut
                   Remove the desktop global shortcut installed above
  clipd help       Print this help

The toggle/show/hide commands are the command-line alternative to the
global Super+V hotkey — bind "clipd toggle" to any shortcut your desktop
(or, under WSL, Windows) provides. On Wayland the X11 hotkey grab can't
work, so clipd auto-installs the shortcut on GNOME; use install-shortcut
for other desktops or to rebind it.
`)
}

func (a *appWiring) onBeforeClose(ctx context.Context) bool {
	// A real quit (tray "Quit" or `clipd quit`) sets the flag — let the close
	// proceed. Otherwise this is a window-close (the X button), which we turn
	// into hide-to-tray so the app keeps running in the background.
	if a.quitting.Load() {
		return false
	}
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
