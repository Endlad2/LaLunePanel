// Package protocol defines the VPN protocols LaLune Panel can deploy.
//
// Role handling:
//   - olcRTC has two distinct sides (srv / cnc) with different config schemas.
//     HasRoles=true, uses CommonFields + FieldsServer + FieldsClient.
//   - OpenFlux is ALWAYS an exit node. HasRoles=false, only CommonFields.
//     The generator hardcodes --role=exit; the UI never asks for a role.
//     No inbound / local_ip — those are not part of this deploy path.
package protocol

import (
	"fmt"
	"runtime"
)

// Role is which side of the tunnel we're configuring.
// Only meaningful for protocols with HasRoles=true (olcRTC).
type Role string

const (
	RoleServer Role = "srv"
	RoleClient Role = "cnc"
)

// Field describes one input the UI must show when creating a client.
type Field struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // text | number | select | password | bool
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Help        string   `json:"help,omitempty"`
	// ShowIf: render this field only if params[ShowIf.Key] == ShowIf.Value.
	// The sentinel "__nonempty__" matches any non-empty value.
	ShowIf *ShowIf `json:"show_if,omitempty"`
}

type ShowIf struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// BinarySource maps an architecture to a download URL.
type BinarySource struct {
	Binary string `json:"binary"`
	AMD64  string `json:"-"`
	ARM64  string `json:"-"`
}

// Protocol is a deployable VPN protocol.
type Protocol struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	HasRoles     bool         `json:"has_roles"`
	CommonFields []Field      `json:"common_fields"`
	FieldsServer []Field      `json:"fields_server,omitempty"`
	FieldsClient []Field      `json:"fields_client,omitempty"`
	Binary       BinarySource `json:"-"`
}

// ConfigResult is what GenerateConfig returns.
type ConfigResult struct {
	Config    string `json:"config"`
	URI       string `json:"uri,omitempty"`
	SocksAddr string `json:"socks_addr,omitempty"`
}

// FieldsFor returns the field list for a given role.
// For protocols without roles (HasRoles=false), role is ignored.
func (p Protocol) FieldsFor(role Role) []Field {
	if !p.HasRoles {
		return p.CommonFields
	}
	out := make([]Field, 0, len(p.CommonFields)+8)
	out = append(out, p.CommonFields...)
	if role == RoleClient {
		out = append(out, p.FieldsClient...)
	} else {
		out = append(out, p.FieldsServer...)
	}
	return out
}

var registry = map[string]Protocol{
	// -------------------------------------------------------------------------
	// olcRTC — two distinct sides (srv / cnc)
	// -------------------------------------------------------------------------
	"olcrtc": {
		ID:          "olcrtc",
		Name:        "olcRTC",
		Description: "WebRTC-tunneled VPN. Server = srv, client = cnc. No inbound ports.",
		HasRoles:    true,
		CommonFields: []Field{
			{Key: "provider", Label: "Provider", Type: "select",
				Options: []string{"jitsi", "telemost", "wbstream"}, Default: "jitsi", Required: true},
			{Key: "transport", Label: "Transport", Type: "select",
				Options: []string{"datachannel", "videochannel", "seichannel", "vp8channel"},
				Default: "datachannel", Required: true},
			{Key: "room", Label: "Room ID / URL", Type: "text", Required: true,
				Placeholder: "https://meet.jit.si/olcrtc-xxxx"},
			{Key: "key", Label: "Encryption key (64 hex)", Type: "text", Required: false,
				Help: "Server: leave blank to auto-generate. Client: paste the server's key."},
			{Key: "debug", Label: "Debug logging", Type: "select",
				Options: []string{"true", "false"}, Default: "true", Required: false},
		},
		FieldsServer: []Field{
			{Key: "dns", Label: "DNS", Type: "text", Default: "8.8.8.8:53", Required: false},
			{Key: "wb_token", Label: "wbstream auth token", Type: "password", Required: false,
				Help: "Only for wbstream provider."},
			{Key: "use_proxy", Label: "Use SOCKS5 egress", Type: "bool", Default: "false", Required: false},
			{Key: "proxy_addr", Label: "SOCKS5 proxy address", Type: "text", Default: "127.0.0.1",
				Required: false, ShowIf: &ShowIf{Key: "use_proxy", Value: "true"}},
			{Key: "proxy_port", Label: "SOCKS5 proxy port", Type: "number", Default: "1080",
				Required: false, ShowIf: &ShowIf{Key: "use_proxy", Value: "true"}},
		},
		FieldsClient: []Field{
			{Key: "socks_ip", Label: "SOCKS5 listen IP", Type: "text", Default: "127.0.0.1", Required: true},
			{Key: "socks_port", Label: "SOCKS5 listen port", Type: "number", Default: "8808", Required: true},
			{Key: "socks_user", Label: "SOCKS5 username", Type: "text", Required: false,
				Help: "Leave empty to disable auth. Required if binding outside loopback."},
			{Key: "socks_pass", Label: "SOCKS5 password", Type: "password", Required: false,
				ShowIf: &ShowIf{Key: "socks_user", Value: "__nonempty__"}},
		},
		Binary: BinarySource{
			Binary: "olcrtc",
			AMD64:  "https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/olcrtc-linux-amd64",
			ARM64:  "https://github.com/Endlad2/LaLunePanelServers/releases/latest/download/olcrtc-linux-arm64",
		},
	},

	// -------------------------------------------------------------------------
	// OpenFlux — always an exit node (--role=exit).
	// No inbound / no local_ip: not part of this deploy path.
	// -------------------------------------------------------------------------
	"openflux": {
		ID:          "openflux",
		Name:        "OpenFlux",
		Description: "TCP tunnel exit node (always --role=exit). Pluggable transports, batched zstd codec.",
		HasRoles:    false,
		CommonFields: []Field{
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
