package clipboard

// Server identifies the platform clipd is running under. This is a
// Windows-only build, so there is exactly one value.
type Server int

const ServerWindows Server = iota

func (s Server) String() string { return "windows" }

// DetectServer reports the platform. Always ServerWindows in this build.
func DetectServer() Server { return ServerWindows }

// MissingTools reports external clipboard tools that are required but absent.
// Windows uses the built-in Win32 clipboard API, so nothing is ever missing.
func MissingTools(s Server) []string { return nil }
