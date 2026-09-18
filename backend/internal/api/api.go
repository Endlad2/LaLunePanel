// Package api implements the LaLune Panel REST API and protocol orchestration.
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lalune-panel/internal/protocol"
	"lalune-panel/internal/store"
	"lalune-panel/internal/sysinfo"
)

type Server struct {
	store        *store.Store
	dataDir      string
	protocolsDir string
}

func New(st *store.Store, dataDir, protocolsDir string) *Server {
	return &Server{store: st, dataDir: dataDir, protocolsDir: protocolsDir}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/protocols", s.handleProtocols)
	mux.HandleFunc("/api/clients", s.handleClients)
	mux.HandleFunc("/api/clients/", s.handleClientByID)
}

// --- helpers ---------------------------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// --- handlers --------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC(),
		"panel":  "LaLune",
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	clients, _ := s.store.List()
	active := 0
	for _, c := range clients {
		if c.Enabled && c.ExpiresAt.After(time.Now()) {
			active++
		}
	}

	writeJSON(w, 200, map[string]any{
		"host":     sysinfo.Host(),
		"clients":  len(clients),
		"active":   active,
		"uptime_s": sysinfo.UptimeSeconds(),
	})
}

func (s *Server) handleProtocols(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, protocol.All())
}

func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		clients, err := s.store.List()
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if clients == nil {
			clients = []store.Client{}
		}
		writeJSON(w, 200, clients)

	case http.MethodPost:
		s.createClient(w, r)

	default:
		writeErr(w, 405, "method not allowed")
	}
}

func (s *Server) handleClientByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/clients/")
	if id == "" {
		writeErr(w, 400, "missing client id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		c, err := s.store.Get(id)
		if err != nil {
			writeErr(w, 404, "client not found")
			return
		}
		writeJSON(w, 200, c)

	case http.MethodDelete:
		if err := s.store.Delete(id); err != nil {
			writeErr(w, 404, "client not found")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "deleted"})

	default:
		writeErr(w, 405, "method not allowed")
	}
}

// createClient handles POST /api/clients.
//
// Body:
//   {
//     "name": "Alice",
//     "protocol": "olcrtc",
//     "role": "srv" | "cnc",
//     "days": 30,
//     "params": { ...protocol-specific fields... }
//   }
//
// For protocols with HasRoles=false (OpenFlux), "role" is forced to "srv".
func (s *Server) createClient(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string            `json:"name"`
		Protocol string            `json:"protocol"`
		Role     string            `json:"role"`
		Days     int               `json:"days"`
		Params   map[string]string `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		writeErr(w, 400, "name is required")
		return
	}
	if req.Days <= 0 {
		req.Days = 30
	}

	proto, ok := protocol.Get(req.Protocol)
	if !ok {
		writeErr(w, 400, "unknown protocol: "+req.Protocol)
		return
	}

	// Resolve role. Protocols without roles are always server-side.
	role := protocol.RoleServer
	if proto.HasRoles {
		role = protocol.Role(req.Role)
		if role != protocol.RoleServer && role != protocol.RoleClient {
			writeErr(w, 400, "role must be 'srv' or 'cnc'")
			return
		}
	}

	if req.Params == nil {
		req.Params = map[string]string{}
	}

	// 1. ensure the server binary for this protocol is present
	binPath, err := protocol.EnsureBinary(s.dataDir, proto)
	if err != nil {
		writeErr(w, 500, "binary download failed: "+err.Error())
		return
	}

	// 2. generate config from user-supplied params
	res, err := protocol.GenerateConfig(proto, role, binPath, req.Params)
	if err != nil {
		writeErr(w, 400, "config generation failed: "+err.Error())
		return
	}

	// 3. persist client
	id := newID()
	cfgDir := filepath.Join(s.dataDir, "clients")
	_ = os.MkdirAll(cfgDir, 0o755)
	cfgPath := filepath.Join(cfgDir, id+".conf")
	if err := os.WriteFile(cfgPath, []byte(res.Config), 0o600); err != nil {
		writeErr(w, 500, "write config: "+err.Error())
		return
	}

	c := &store.Client{
		ID:         id,
		Name:       req.Name,
		Protocol:   req.Protocol,
		Role:       string(role),
		Config:     res.Config,
		ConfigFile: fmt.Sprintf("clients/%s.conf", id),
		URI:        res.URI,
		SocksAddr:  res.SocksAddr,
		CreatedAt:  time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().AddDate(0, 0, req.Days),
		Enabled:    true,
	}
	if err := s.store.Save(c); err != nil {
		writeErr(w, 500, "save client: "+err.Error())
		return
	}

	writeJSON(w, 201, c)
}
