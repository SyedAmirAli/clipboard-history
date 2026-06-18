package clipboard

// ContentType represents the kind of clipboard content stored.
type ContentType string

const (
	ContentTypeText  ContentType = "text"
	ContentTypeImage ContentType = "image"
)

// Item is a single clipboard history entry.
// The JSON shape is what gets sent to the frontend, so we deliberately
// omit the heavy ImageBlob field from the wire payload — only the small
// ImageThumb (base64 PNG data URL) is sent. Full image bytes are only
// loaded when the user pastes that item.
type Item struct {
	ID          int64       `json:"id"`
	ContentType ContentType `json:"contentType"`
	TextContent string      `json:"textContent,omitempty"`
	Preview     string      `json:"preview"`
	ImageThumb  string      `json:"imageThumb,omitempty"`
	ImageW      int         `json:"imageW,omitempty"`
	ImageH      int         `json:"imageH,omitempty"`
	ContentHash string      `json:"contentHash"`
	Pinned      bool        `json:"pinned"`
	CreatedAt   int64       `json:"createdAt"`
	LastUsedAt  int64       `json:"lastUsedAt"`
}

// Settings represents user-configurable application settings.
type Settings struct {
	Hotkey      string `json:"hotkey"`
	MaxItems    int    `json:"maxItems"`
	KeepImages  bool   `json:"keepImages"`
	MaxImageMB  int    `json:"maxImageMB"`
	Theme       string `json:"theme"`
	Autostart   bool   `json:"autostart"`
	HideOnBlur  bool   `json:"hideOnBlur"`
	LaunchAtTop bool   `json:"launchAtTop"`
	// WindowFrame, when true, runs clipd as a normal decorated window with
	// native minimize/maximize/close buttons and a taskbar entry. When
	// false, clipd is a frameless always-on-top popup. Applied at startup
	// (changing it requires a restart).
	WindowFrame bool `json:"windowFrame"`
	// AutoPaste, when true, synthesises a Ctrl+V into the previously focused
	// window after an item is selected, so the value is pasted automatically.
	AutoPaste bool `json:"autoPaste"`
	// ShowInTaskbar, when true, gives the window a normal taskbar button (so it
	// minimises to the taskbar). When false, the window is a taskbar-less popup.
	// Applied live.
	ShowInTaskbar bool `json:"showInTaskbar"`
}

// DefaultSettings returns platform-specific default settings.
// Implemented in types_windows.go and types_linux.go
