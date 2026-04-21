package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultFactorioImage = "factoriotools/factorio"
	DefaultImageTag      = "stable"
	DefaultGamePort      = 34197
	DefaultRCONPort      = 27015
)

// AppConfig is the global application configuration stored in config.json.
type AppConfig struct {
	FactorioUsername string `json:"factorio_username,omitempty"`
	FactorioToken    string `json:"factorio_token,omitempty"`
	DefaultImageTag  string `json:"default_image_tag,omitempty"`
}

// ServerStatus represents the runtime status of a server.
type ServerStatus string

const (
	StatusRunning ServerStatus = "running"
	StatusStopped ServerStatus = "stopped"
	StatusCreated ServerStatus = "created"
	StatusUnknown ServerStatus = "unknown"
)

// ServerConfig holds the configuration for a single Factorio server instance.
type ServerConfig struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ImageTag    string    `json:"image_tag"`
	ContainerID string    `json:"container_id,omitempty"`
	GamePort    int       `json:"game_port"`
	RCONPort    int       `json:"rcon_port"`
	RCONPassword string   `json:"rcon_password,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ServersRegistry is the list of all managed servers stored in servers.json.
type ServersRegistry struct {
	Servers []*ServerConfig `json:"servers"`
}

// Manager handles reading and writing of all configuration files.
type Manager struct {
	dataDir string
}

// NewManager creates a new config Manager using the given data directory.
func NewManager(dataDir string) *Manager {
	return &Manager{dataDir: dataDir}
}

// DataDir returns the base data directory.
func (m *Manager) DataDir() string {
	return m.dataDir
}

// ServerDir returns the data directory for a specific server.
func (m *Manager) ServerDir(name string) string {
	return filepath.Join(m.dataDir, "servers", name)
}

// ConfigDir returns the config sub-directory for a specific server.
func (m *Manager) ConfigDir(name string) string {
	return filepath.Join(m.ServerDir(name), "config")
}

// ModsDir returns the mods sub-directory for a specific server.
func (m *Manager) ModsDir(name string) string {
	return filepath.Join(m.ServerDir(name), "mods")
}

// SavesDir returns the saves sub-directory for a specific server.
func (m *Manager) SavesDir(name string) string {
	return filepath.Join(m.ServerDir(name), "saves")
}

// Init ensures the data directory and base files exist.
func (m *Manager) Init() error {
	if err := os.MkdirAll(m.dataDir, 0700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Create config.json if missing
	cfgPath := filepath.Join(m.dataDir, "config.json")
	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		defaults := &AppConfig{DefaultImageTag: DefaultImageTag}
		if err := writeJSON(cfgPath, defaults); err != nil {
			return fmt.Errorf("init config.json: %w", err)
		}
	}

	// Create servers.json if missing
	regPath := filepath.Join(m.dataDir, "servers.json")
	if _, err := os.Stat(regPath); errors.Is(err, os.ErrNotExist) {
		if err := writeJSON(regPath, &ServersRegistry{Servers: []*ServerConfig{}}); err != nil {
			return fmt.Errorf("init servers.json: %w", err)
		}
	}

	return nil
}

// LoadAppConfig reads and returns the global app config.
func (m *Manager) LoadAppConfig() (*AppConfig, error) {
	cfg := &AppConfig{}
	if err := readJSON(filepath.Join(m.dataDir, "config.json"), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SaveAppConfig writes the global app config.
func (m *Manager) SaveAppConfig(cfg *AppConfig) error {
	return writeJSON(filepath.Join(m.dataDir, "config.json"), cfg)
}

// LoadRegistry reads and returns the servers registry.
func (m *Manager) LoadRegistry() (*ServersRegistry, error) {
	reg := &ServersRegistry{}
	if err := readJSON(filepath.Join(m.dataDir, "servers.json"), reg); err != nil {
		return nil, err
	}
	if reg.Servers == nil {
		reg.Servers = []*ServerConfig{}
	}
	return reg, nil
}

// SaveRegistry writes the servers registry.
func (m *Manager) SaveRegistry(reg *ServersRegistry) error {
	return writeJSON(filepath.Join(m.dataDir, "servers.json"), reg)
}

// GetServer finds a server by name in the registry.
func (m *Manager) GetServer(name string) (*ServerConfig, error) {
	reg, err := m.LoadRegistry()
	if err != nil {
		return nil, err
	}
	for _, s := range reg.Servers {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("server %q not found", name)
}

// AddServer adds a server to the registry.
func (m *Manager) AddServer(srv *ServerConfig) error {
	reg, err := m.LoadRegistry()
	if err != nil {
		return err
	}
	for _, s := range reg.Servers {
		if s.Name == srv.Name {
			return fmt.Errorf("server %q already exists", srv.Name)
		}
	}
	reg.Servers = append(reg.Servers, srv)
	return m.SaveRegistry(reg)
}

// UpdateServer updates an existing server in the registry.
func (m *Manager) UpdateServer(srv *ServerConfig) error {
	reg, err := m.LoadRegistry()
	if err != nil {
		return err
	}
	for i, s := range reg.Servers {
		if s.Name == srv.Name {
			srv.UpdatedAt = time.Now()
			reg.Servers[i] = srv
			return m.SaveRegistry(reg)
		}
	}
	return fmt.Errorf("server %q not found", srv.Name)
}

// RemoveServer removes a server from the registry by name.
func (m *Manager) RemoveServer(name string) error {
	reg, err := m.LoadRegistry()
	if err != nil {
		return err
	}
	filtered := make([]*ServerConfig, 0, len(reg.Servers))
	found := false
	for _, s := range reg.Servers {
		if s.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, s)
	}
	if !found {
		return fmt.Errorf("server %q not found", name)
	}
	reg.Servers = filtered
	return m.SaveRegistry(reg)
}

// InitServerDirs creates the directories for a server instance.
func (m *Manager) InitServerDirs(name string) error {
	for _, dir := range []string{
		m.ConfigDir(name),
		m.ModsDir(name),
		m.SavesDir(name),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}
	return nil
}

// RemoveServerDirs deletes all data directories for a server instance.
func (m *Manager) RemoveServerDirs(name string) error {
	return os.RemoveAll(m.ServerDir(name))
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
