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
}

// DefaultSettings returns the baseline settings used on first run.
func DefaultSettings() Settings {
	return Settings{
		Hotkey:      "Super+V",
		MaxItems:    200,
		KeepImages:  true,
		MaxImageMB:  5,
		Theme:       "auto",
		Autostart:   false,
		HideOnBlur:  true,
		LaunchAtTop: false,
	}
}
