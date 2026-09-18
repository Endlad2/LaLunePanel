// Package store persists clients to disk as JSON files.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Client is one VPN client record.
type Client struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Protocol    string    `json:"protocol"`     // "olcrtc" | "openflux"
	Config      string    `json:"config"`       // generated config content
	ConfigFile  string    `json:"config_file"`  // path relative to data dir
	URI         string    `json:"uri,omitempty"`// olcrtc:// or similar
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Enabled     bool      `json:"enabled"`
}

type Store struct {
	dir string
	mu  sync.RWMutex
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *Store) List() ([]Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var out []Client
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var c Client
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}
		out = append(out, c)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Store) Get(id string) (*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var c Client
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) Save(c *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c.ID == "" {
		return errors.New("client id is empty")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(c.ID), data, 0o600)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "" {
		return fmt.Errorf("client id is empty")
	}
	return os.Remove(s.path(id))
}
