// Package protocol defines the VPN protocols LaLune Panel can deploy.
//
// Each protocol knows:
//   - which fields the UI must collect
//   - how to download its server binary
//   - how to turn collected fields into a config file + shareable URI
package protocol

import (
	"fmt"
	"runtime"
)

// Field describes one input the UI must show when creating a client.
type Field struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // text | number | select | password
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Help        string   `json:"help,omitempty"`
}

// BinarySource maps an architecture to a download URL.
type BinarySource struct {
	Binary string `json:"binary"` // filename as stored in /data/bin
	AMD64  string `json:"-"`
	ARM64  string `json:"-"`
}

// Protocol is a deployable VPN protocol.
type Protocol struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Fields      []Field      `json:"fields"`
	Binary      BinarySource `json:"-"`
	Mode        string       `json:"mode"` // "server" — these are server-side daemons
}

// ConfigResult is what GenerateConfig returns.
type ConfigResult struct {
	Config string `json:"config"`
	URI    string `json:"uri,omitempty"`
}

var registry = map[string]Protocol{
	"olcrtc": {
		ID:          "olcrtc",
		Name:        "olcRTC",
		Description: "WebRTC-tunneled VPN. Server runs as 'srv', client is 'cnc'.",
		Mode:        "server",
		Fields: []Field{
			{Key: "provider", Label: "Provider", Type: "select",
				Options: []string{"jitsi", "telemost", "wbstream"}, Default: "jitsi", Required: true},
			{Key: "transport", Label: "Transport", Type: "select",
				Options: []string{"datachannel", "videochannel", "seichannel", "vp8channel"},
				Default: "datachannel", Required: true},
			{Key: "room", Label: "Room ID / URL", Type: "text", Required: true,
				Placeholder: "https://meet.jit.si/olcrtc-xxxx"},
			{Key: "key", Label: "Encryption key (64 hex)", Type: "text", Required: true,
				Help: "Leave blank on server to auto-generate; client must match."},
			{Key: "dns", Label: "DNS", Type: "text", Default: "8.8.8.8:53", Required: false},
			{Key: "wb_token", Label: "wbstream auth token", Type: "password", Required: false,
				Help: "Only for wbstream provider."},
		},
		Binary: BinarySource{
			Binary: "olcrtc",
			AMD64:  "https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/olcrtc-linux-amd64",
			ARM64:  "https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/olcrtc-linux-arm64",
		},
	},

	"openflux": {
		ID:          "openflux",
		Name:        "OpenFlux",
		Description: "TCP tunnel with pluggable transports and batched zstd codec.",
		Mode:        "server",
		Fields: []Field{
			{Key: "mode", Label: "Exit mode", Type: "select",
				Options: []string{"l3", "l4"}, Default: "l4", Required: true,
				Help: "l3 needs root + Linux; l4 works everywhere."},
			{Key: "transport", Label: "Transport", Type: "select",
				Options: []string{"yandex", "vyandex", "oneme", "cupsonline", "mailru"},
				Default: "yandex", Required: true},
			{Key: "url", Label: "Transport URL", Type: "text", Required: true,
				Placeholder: "YOUR_YANDEX_DOC_URL"},
			{Key: "codec", Label: "Codec", Type: "select",
				Options: []string{"batched", "legacy"}, Default: "batched", Required: false},
			{Key: "encryption_key_file", Label: "Encryption key file", Type: "text", Required: false,
				Help: "Optional path to a shared AES-256-GCM secret."},
			{Key: "max_token", Label: "MAX token", Type: "password", Required: false,
				Help: "Only for --transport=oneme."},
			{Key: "max_uid", Label: "MAX UID", Type: "text", Required: false,
				Help: "Only for --transport=oneme."},
			{Key: "debug", Label: "Debug logging", Type: "select",
				Options: []string{"false", "true"}, Default: "false", Required: false},
		},
		Binary: BinarySource{
			Binary: "openflux",
			AMD64:  "https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/OpenFlux-linux-amd64",
			ARM64:  "https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/OpenFlux-linux-arm64",
		},
	},
}

// All returns every registered protocol for the UI.
func All() []Protocol {
	out := make([]Protocol, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}

// Get looks up a protocol by ID.
func Get(id string) (Protocol, bool) {
	p, ok := registry[id]
	return p, ok
}

// URLForArch picks the right download URL for the current platform.
func (p Protocol) URLForArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		if p.Binary.AMD64 == "" {
			return "", fmt.Errorf("no amd64 binary for %s", p.ID)
		}
		return p.Binary.AMD64, nil
	case "arm64":
		if p.Binary.ARM64 == "" {
			return "", fmt.Errorf("no arm64 binary for %s", p.ID)
		}
		return p.Binary.ARM64, nil
	default:
		return "", fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
	}
}
