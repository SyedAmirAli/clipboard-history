package config

import (
	"os"
	"path/filepath"
)

// AppName is used to build XDG-compatible paths.
const AppName = "clipd"

// DataDir returns the directory where clipd stores its SQLite database
// and other persistent state. Honours $XDG_DATA_HOME, falling back to
// ~/.local/share/<AppName>. The directory is created if it doesn't exist.
func DataDir() (string, error) {
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

// ConfigDir returns the directory for user configuration following XDG
// conventions. Honours $XDG_CONFIG_HOME, falling back to ~/.config/<AppName>.
func ConfigDir() (string, error) {
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

// AutostartDir returns the freedesktop autostart directory under
// $XDG_CONFIG_HOME/autostart. The directory is created if it doesn't exist.
func AutostartDir() (string, error) {
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

// SocketPath returns the path to the Unix domain socket used for
// single-instance control (e.g. `clipd toggle`).
//
// It lives in the data dir rather than $XDG_RUNTIME_DIR on purpose: the
// path must resolve identically whether clipd is launched from a normal
// login session (which sets XDG_RUNTIME_DIR) or from a bare, env-less
// invocation such as `wsl.exe clipd toggle` from a Windows hotkey. Keying
// it off the data dir (derived from $HOME) keeps the daemon and any client
// in agreement regardless of how each was started.
func SocketPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName+".sock"), nil
}
