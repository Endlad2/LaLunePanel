package protocol

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// GenerateConfig turns UI-supplied params into a config file.
//
// role is only meaningful for protocols with HasRoles=true (olcRTC).
// For OpenFlux, role is ignored — the exit-node role is hardcoded.
func GenerateConfig(p Protocol, role Role, binPath string, params map[string]string) (*ConfigResult, error) {
	fields := p.FieldsFor(role)

	// fill defaults
	for _, f := range fields {
		if _, ok := params[f.Key]; !ok && f.Default != "" {
			params[f.Key] = f.Default
		}
	}

	// validate required (skip fields hidden by ShowIf)
	for _, f := range fields {
		if !f.Required {
			continue
		}
		if f.ShowIf != nil && !showIfSatisfied(f.ShowIf, params) {
			continue
		}
		if strings.TrimSpace(params[f.Key]) == "" {
			return nil, fmt.Errorf("field %q is required", f.Label)
		}
	}

	switch p.ID {
	case "olcrtc":
		return genOlcRTC(role, params)
	case "openflux":
		return genOpenFlux(params)
	default:
		return nil, fmt.Errorf("no config generator for %s", p.ID)
	}
}

// showIfSatisfied evaluates a ShowIf predicate against params.
// The sentinel "__nonempty__" means "any non-empty value".
func showIfSatisfied(s *ShowIf, params map[string]string) bool {
	v := params[s.Key]
	if s.Value == "__nonempty__" {
		return strings.TrimSpace(v) != ""
	}
	return v == s.Value
}

// --- olcRTC -----------------------------------------------------------------

func genOlcRTC(role Role, params map[string]string) (*ConfigResult, error) {
	key := strings.TrimSpace(params["key"])
	if key == "" {
		if role == RoleClient {
			return nil, fmt.Errorf("encryption key is required for client")
		}
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		key = hex.EncodeToString(b)
	}

	provider := strings.TrimSpace(params["provider"])
	transport := strings.TrimSpace(params["transport"])
	room := strings.TrimSpace(params["room"])
	debug := params["debug"]
	if debug == "" {
		debug = "false"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "mode: %s\n", role)
	fmt.Fprintf(&b, "auth:\n")
	fmt.Fprintf(&b, "  provider: %q\n", provider)

	if t := strings.TrimSpace(params["wb_token"]); t != "" && provider == "wbstream" {
		fmt.Fprintf(&b, "  token: %q\n", t)
	}

	fmt.Fprintf(&b, "room:\n")
	fmt.Fprintf(&b, "  id: %q\n", room)
	fmt.Fprintf(&b, "crypto:\n")
	fmt.Fprintf(&b, "  key: %q\n", key)
	fmt.Fprintf(&b, "net:\n")
	fmt.Fprintf(&b, "  transport: %q\n", transport)

	if role == RoleServer {
		dns := strings.TrimSpace(params["dns"])
		if dns == "" {
			dns = "8.8.8.8:53"
		}
		fmt.Fprintf(&b, "  dns: %q\n", dns)

		if params["use_proxy"] == "true" {
			addr := strings.TrimSpace(params["proxy_addr"])
			if addr == "" {
				addr = "127.0.0.1"
			}
			port := strings.TrimSpace(params["proxy_port"])
			if port == "" {
				port = "1080"
			}
			fmt.Fprintf(&b, "socks:\n")
			fmt.Fprintf(&b, "  proxy_addr: %q\n", addr)
			fmt.Fprintf(&b, "  proxy_port: %s\n", port)
		}
	} else {
		ip := strings.TrimSpace(params["socks_ip"])
		if ip == "" {
			ip = "127.0.0.1"
		}
		port := strings.TrimSpace(params["socks_port"])
		if port == "" {
			port = "8808"
		}
		fmt.Fprintf(&b, "socks:\n")
		fmt.Fprintf(&b, "  host: %q\n", ip)
		fmt.Fprintf(&b, "  port: %s\n", port)

		user := strings.TrimSpace(params["socks_user"])
		pass := strings.TrimSpace(params["socks_pass"])
		if user != "" {
			fmt.Fprintf(&b, "  user: %q\n", user)
			fmt.Fprintf(&b, "  pass: %q\n", pass)
		}
	}

	fmt.Fprintf(&b, "debug: %s\n", debug)

	uri := fmt.Sprintf("olcrtc://%s?%s@%s#%s", provider, transport, room, key)

	res := &ConfigResult{Config: b.String(), URI: uri}
	if role == RoleClient {
		ip := strings.TrimSpace(params["socks_ip"])
		if ip == "" {
			ip = "127.0.0.1"
		}
		port := strings.TrimSpace(params["socks_port"])
		if port == "" {
			port = "8808"
		}
		res.SocksAddr = fmt.Sprintf("%s:%s", ip, port)
	}
	return res, nil
}

// --- OpenFlux ---------------------------------------------------------------
//
// OpenFlux in LaLune is always an exit node. --role=exit is hardcoded.
// No inbound / no local_ip.

func genOpenFlux(params map[string]string) (*ConfigResult, error) {
	transport := strings.TrimSpace(params["transport"])
	url := strings.TrimSpace(params["url"])
	codec := strings.TrimSpace(params["codec"])
	if codec == "" {
		codec = "batched"
	}

	args := []string{
		"--role=exit",
		"--transport=" + transport,
		"--url=" + url,
	}

	if codec != "batched" {
		args = append(args, "--codec="+codec)
	}
	if f := strings.TrimSpace(params["encryption_key_file"]); f != "" {
		args = append(args, "--encryption-key-file="+f)
	}
	if t := strings.TrimSpace(params["max_token"]); t != "" {
		args = append(args, "--maxToken="+t)
	}
	if u := strings.TrimSpace(params["max_uid"]); u != "" {
		args = append(args, "--maxUid="+u)
	}
	if params["debug"] == "true" {
		args = append(args, "--debug")
	}

	cmd := "./openflux " + strings.Join(args, " ")

	var out strings.Builder
	fmt.Fprintf(&out, "# OpenFlux exit node\n")
	fmt.Fprintf(&out, "# Generated by LaLune Panel\n")
	fmt.Fprintf(&out, "command: %s\n", cmd)
	fmt.Fprintf(&out, "role: exit\n")
	fmt.Fprintf(&out, "transport: %s\n", transport)
	fmt.Fprintf(&out, "url: %s\n", url)
	fmt.Fprintf(&out, "codec: %s\n", codec)

	return &ConfigResult{Config: out.String()}, nil
}
