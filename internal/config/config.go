package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// AppName is used to build config paths.
const AppName = "clipd"

// DataDir returns the directory where clipd stores its SQLite database
// and other persistent state. On Windows: %APPDATA%\clipd, on Linux: ~/.local/share/clipd
func DataDir() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		dir := filepath.Join(appData, AppName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		return dir, nil
	}
	// Linux: XDG conventions
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, AppName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ConfigDir returns the directory for user configuration. On Windows: %APPDATA%\clipd,
// on Linux: ~/.config/clipd following XDG conventions.
func ConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		dir := filepath.Join(appData, AppName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		return dir, nil
	}
	// Linux: XDG conventions
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, AppName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// AutostartDir returns the autostart directory. On Windows: data dir (registry used instead).
// On Linux: ~/.config/autostart following freedesktop conventions.
func AutostartDir() (string, error) {
	if runtime.GOOS == "windows" {
		return DataDir() // Use same dir; Windows uses registry, not files
	}
	// Linux: freedesktop
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "autostart")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// DBPath returns the absolute path to the SQLite database file.
func DBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "clipd.db"), nil
}

// AutostartFilePath returns the path to the user's autostart .desktop entry.
func AutostartFilePath() (string, error) {
	dir, err := AutostartDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName+".desktop"), nil
}

// ControlChannelPath returns the path for IPC communication:
// - Windows: named pipe path "\\\\.\\pipe\\clipd"
// - Linux: Unix socket path in the data dir
func ControlChannelPath() (string, error) {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\clipd`, nil
	}
	// Linux: Unix socket
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName+".sock"), nil
}

// SocketPath is deprecated; use ControlChannelPath instead.
// Kept for backwards compatibility — maps to ControlChannelPath on Windows,
// the old Unix socket path on Linux.
func SocketPath() (string, error) {
	return ControlChannelPath()
}
