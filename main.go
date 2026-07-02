package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"clipd/internal/config"
	"clipd/internal/db"
	"clipd/internal/ipc"
	"clipd/internal/service"
	"clipd/internal/shortcut"
	"clipd/internal/tray"
	"clipd/internal/watcher"
	"clipd/internal/x11hint"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

// version is the clipd release, injected at build time via
//
//	-ldflags "-X main.version=<v>"
//
// (scripts/build.sh reads it from wails.json so there's a single source of
// truth). It falls back to "dev" for plain `go build` / `go run`.
var version = "dev"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Must run before Wails creates the webview: webkit2gtk reads these env
	// vars only at WebView init time.
	hardenLinuxWebView()

	// GNOME shows a "<app> is ready" notification (and blinks the dock icon)
	// when a startup-activation sequence completes without presenting a window.
	// Drop any inherited activation token once at startup so repeated window
	// presents do not re-trigger those notifications.
	os.Unsetenv("DESKTOP_STARTUP_ID")
	os.Unsetenv("XDG_ACTIVATION_TOKEN")

	sockPath, _ := config.SocketPath()

	// CLI control mode: `clipd toggle|show|hide` forwards the command to a
	// running instance and exits. The desktop shortcut (GNOME gsettings)
	// invokes `clipd toggle`; this is also handy for users who prefer to
	// bind the command themselves on other desktops.
	explicitStart := false
	if len(os.Args) > 1 {
		// Version is handled before anything that needs the control socket so it
		// works even when the socket path can't be resolved.
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("clipd %s\n", version)
			return
		}
		if sockPath == "" {
			log.Fatal("clipd: cannot resolve control socket path")
		}
		switch arg := os.Args[1]; arg {
		case "start":
			// Explicit start: fall through to GUI startup below. Unlike a
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

	// The GUI process does NOT watch the clipboard — that runs in the separate
	// clipd-watch script so this process stays completely idle while hidden,
	// which is what eliminates the dock/taskbar flicker. The GUI only reads the
	// DB the worker seeds and refreshes when shown.
	//
	// The global shortcut is a desktop binding (GNOME gsettings) that launches
	// or toggles this GUI; shortcut.Manager is the HotkeySetter the service
	// re-installs on change.
	sc := shortcut.NewManager()

	svc := service.New(store, sc)

	// The window is always frameless — the frontend draws its own title bar
	// (macOS-style traffic lights). The WindowFrame setting selects behaviour:
	// windowed mode = a normal taskbar window; popup mode = an always-on-top
	// Windows+V style overlay that hides on focus loss.
	windowed := svc.CurrentSettings().WindowFrame

	// A translucent (ARGB) window flickers continuously on the NVIDIA
	// proprietary driver under XWayland — the web process stays alive and the
	// log is clean, but the transparent surface strobes ("display blinking").
	// On NVIDIA we make the window opaque: the only cost is that the area
	// outside the frontend's rounded corners shows the dark background colour
	// instead of being see-through, which is barely visible on the dark theme.
	nvidia := hasNvidiaDriver()
	bgAlpha := uint8(0)
	if nvidia {
		bgAlpha = 255
	}

	app := &appWiring{
		svc:      svc,
		sockPath: sockPath,
		// Bare `clipd` (what the hotkey runs) means "open the popup", so show it
		// on launch. `clipd start` launches the GUI resident/hidden.
		showOnStart: !explicitStart,
	}

	err = wails.Run(&options.App{
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
		BackgroundColour:  &options.RGBA{R: 22, G: 23, B: 30, A: bgAlpha},
		AssetServer:       &assetserver.Options{Assets: assets},
		OnStartup:         app.onStartup,
		OnShutdown:        app.onShutdown,
		OnBeforeClose:     app.onBeforeClose,
		Bind:              []any{svc},
		Linux: &linux.Options{
			Icon: appIcon,
			// Translucent so the area outside the frontend's rounded corners is
			// transparent — but NOT on NVIDIA, where ARGB windows strobe under
			// XWayland (see bgAlpha above). Opaque there trades the see-through
			// corners for a stable, non-blinking window.
			WindowIsTranslucent: !nvidia,
			// OnDemand (not Always): forcing GPU acceleration even when the
			// driver is blacklisted is what crash-loops the webkit web process
			// on Wayland — blank/black window, blinking taskbar icon, and a
			// dead IPC socket (so the shortcut stops responding). OnDemand lets
			// webkit fall back to software rendering on flaky GPUs.
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
			ProgramName:      "clipd",
		},
	})
	if err != nil {
		log.Fatalf("wails run: %v", err)
	}
}

// hardenLinuxWebView sets webkit2gtk environment variables that keep the
// embedded WebView stable on Linux — above all under Wayland, where the
// default DMABUF-based renderer crashes or paints a blank/black window on many
// Intel/AMD/NVIDIA driver combinations. The visible symptoms are flickering, a
// blinking taskbar icon (the web process restarting in a loop), and the app
// "dying" — when the renderer dies it takes the window and the control socket
// with it, so the global shortcut appears broken too.
//
// Each var is only set when the user hasn't already chosen a value, so a power
// user can still override (e.g. re-enable the DMABUF renderer, or also set
// WEBKIT_DISABLE_COMPOSITING_MODE=1 as a heavier fallback if a blank window
// persists — we don't set that one by default because it disables the GPU
// compositing the translucent rounded corners rely on).
func hardenLinuxWebView() {
	setIfUnset := func(k, v string) {
		if _, ok := os.LookupEnv(k); !ok {
			_ = os.Setenv(k, v)
		}
	}
	// Disable the DMABUF renderer everywhere: the standard fix for blank/black
	// windows and GPU crashes under webkit2gtk 2.40+ on Intel/AMD Wayland.
	setIfUnset("WEBKIT_DISABLE_DMABUF_RENDERER", "1")

	// NVIDIA's proprietary driver crash-loops inside webkit2gtk on a native
	// Wayland session — the window blinks and the taskbar entry flickers as the
	// web process restarts on a loop. Its GLX/X11 path is far more mature, so on
	// NVIDIA we render this app's window through XWayland and turn off
	// accelerated compositing. Clipboard capture (wl-clipboard) and the GNOME
	// global shortcut stay fully Wayland-native; only clipd's own rendering
	// moves to the stable path. Both are skipped if the user set them already.
	if hasNvidiaDriver() {
		setIfUnset("GDK_BACKEND", "x11")
		setIfUnset("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	}
}

// hasNvidiaDriver reports whether the proprietary NVIDIA kernel driver is
// loaded. We probe a few stable paths it creates rather than shelling out to
// nvidia-smi, so detection is fast and dependency-free.
func hasNvidiaDriver() bool {
	for _, p := range []string{"/proc/driver/nvidia/version", "/dev/nvidiactl", "/sys/module/nvidia"} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
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
	sockPath    string
	showOnStart bool
	ipcLn       *ipc.Listener
	quitting    atomic.Bool
	cleanupOnce sync.Once
}

// cleanup releases all long-lived resources. Safe to call more than once.
func (a *appWiring) cleanup() {
	a.cleanupOnce.Do(func() {
		tray.Quit()
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

	// The clipboard watcher lives in the separate `clipd-watch` process, so
	// this GUI stays idle while hidden (no polling → no dock/taskbar flicker).
	if err := watcher.EnsureEnabled(); err != nil {
		log.Printf("watcher extension: %v", err)
		wailsruntime.LogWarning(ctx, "Clipboard watcher extension is not enabled. "+err.Error())
	}

	// Wayland forbids in-process global key grabs, so register the shortcut
	// with the desktop (GNOME via gsettings) to run `clipd toggle`. On other
	// desktops this logs guidance for binding it manually.
	settings := a.svc.CurrentSettings()
	if err := shortcut.EnsureInstalled(settings.Hotkey); err != nil {
		log.Printf("desktop shortcut: %v", err)
		wailsruntime.LogWarning(ctx, "Global hotkey needs a desktop binding. "+err.Error())
	}

	// Mark the window skip-taskbar while it's still hidden, before its first
	// map — this is what actually stops GNOME's "wl-clipboard is ready" toast
	// and the dock/taskbar blinking (windowAttentionHandler ignores
	// skip-taskbar windows), and removes the taskbar entry entirely.
	// Popup mode only: in windowed ("taskbar window") mode the user wants a
	// taskbar entry, and the skip-taskbar/utility hints also make GNOME refuse
	// to minimise the window (broken yellow button).
	if !settings.WindowFrame {
		x11hint.SuppressTaskbar()
	}

	tray.Start(tray.Callbacks{
		OnOpen:     a.svc.ShowPopup,
		OnClear:    func() { _ = a.svc.ClearAll(true) },
		OnSettings: a.svc.ShowPopup,
		OnQuit:     func() { a.quit(ctx) },
	})

	// A bare `clipd` (the hotkey) means "open the popup now". `clipd start`
	// leaves the GUI hidden and resident until the next hotkey press.
	if a.showOnStart {
		a.svc.ShowPopup()
	}
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
  clipd            Open the clipboard popup (launches the GUI, or toggles it)
  clipd start      Start the GUI (reports if it's already running)
  clipd toggle     Show/hide the clipboard popup of the running GUI
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
  clipd version    Print the clipd version (aliases: -v, --version)
  clipd help       Print this help

clipd is Wayland-only. Since Wayland clients can't grab a global key for
themselves, the global Super+V shortcut is a desktop binding that runs
"clipd toggle"; clipd auto-installs it on GNOME. Use install-shortcut for
other desktops or to rebind it, or bind "clipd toggle" manually.
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
