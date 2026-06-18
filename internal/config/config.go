package config

import (
	"os"
	"path/filepath"
)

// AppName is used to build config paths.
const AppName = "clipd"

// appDataDir returns %APPDATA%\clipd (falling back to ~\AppData\Roaming\clipd),
// creating it if needed. All clipd state lives here.
func appDataDir() (string, error) {
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

// DataDir returns the directory where clipd stores its SQLite database and
// other persistent state: %APPDATA%\clipd.
func DataDir() (string, error) { return appDataDir() }

// ConfigDir returns the directory for user configuration: %APPDATA%\clipd.
func ConfigDir() (string, error) { return appDataDir() }

// DBPath returns the absolute path to the SQLite database file.
func DBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "clipd.db"), nil
}

// ControlChannelPath returns the named-pipe path used for single-instance IPC
// (`clipd toggle|show|hide|quit`).
func ControlChannelPath() (string, error) {
	return `\\.\pipe\clipd`, nil
}

// SocketPath is kept as an alias of ControlChannelPath for callers that still
// use the older name.
func SocketPath() (string, error) {
	return ControlChannelPath()
}
