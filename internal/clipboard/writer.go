package clipboard

// WriteText puts the given UTF-8 string onto the system clipboard.
// Platform-specific implementations in writer_linux.go and writer_windows.go.
func WriteText(s string) error {
	panic("WriteText must be implemented by platform-specific code")
}

// WriteImagePNG puts a PNG image onto the system clipboard.
// Platform-specific implementations in writer_linux.go and writer_windows.go.
func WriteImagePNG(png []byte) error {
	panic("WriteImagePNG must be implemented by platform-specific code")
}
