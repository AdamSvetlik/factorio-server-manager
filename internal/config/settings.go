package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ServerSettings mirrors the structure of Factorio's server-settings.json.
type ServerSettings struct {
	Name                               string   `json:"name"`
	Description                        string   `json:"description"`
	Tags                               []string `json:"tags"`
	MaxPlayers                         int      `json:"max_players"`
	Visibility                         struct {
		Public bool `json:"public"`
		LAN    bool `json:"lan"`
	} `json:"visibility"`
	Username                           string `json:"username"`
	Token                              string `json:"token"`
	GamePassword                       string `json:"game_password"`
	RequireUserVerification            bool   `json:"require_user_verification"`
	MaxUploadInKilobytesPerSecond      int    `json:"max_upload_in_kilobytes_per_second"`
	MaxUploadSlots                     int    `json:"max_upload_slots"`
	MinimumLatencyInTicks              int    `json:"minimum_latency_in_ticks"`
	IgnorePlayerLimitForReturningPlayers bool  `json:"ignore_player_limit_for_returning_players"`
	AllowCommands                      string `json:"allow_commands"`
	AutosaveInterval                   int    `json:"autosave_interval"`
	AutosaveSlots                      int    `json:"autosave_slots"`
	AFKAutokickInterval                int    `json:"afk_autokick_interval"`
	AutoPause                          bool   `json:"auto_pause"`
	OnlyAdminsCanPauseTheGame          bool   `json:"only_admins_can_pause_the_game"`
	AutosaveOnlyOnServer               bool   `json:"autosave_only_on_server"`
	NonBlockingSaving                  bool   `json:"non_blocking_saving"`
	MinimumSegmentSize                 int    `json:"minimum_segment_size"`
	MinimumSegmentSizePeerCount        int    `json:"minimum_segment_size_peer_count"`
	MaximumSegmentSize                 int    `json:"maximum_segment_size"`
	MaximumSegmentSizePeerCount        int    `json:"maximum_segment_size_peer_count"`
}

// DefaultServerSettings returns sensible defaults for a new server.
func DefaultServerSettings(name string) *ServerSettings {
	return &ServerSettings{
		Name:                        name,
		Description:                 "",
		Tags:                        []string{},
		MaxPlayers:                  0,
		RequireUserVerification:     true,
		AllowCommands:               "admins-only",
		AutosaveInterval:            10,
		AutosaveSlots:               5,
		AFKAutokickInterval:         0,
		AutoPause:                   true,
		OnlyAdminsCanPauseTheGame:   true,
		AutosaveOnlyOnServer:        true,
		NonBlockingSaving:           false,
		MinimumSegmentSize:          25,
		MinimumSegmentSizePeerCount: 20,
		MaximumSegmentSize:          100,
		MaximumSegmentSizePeerCount: 10,
	}
}

// LoadServerSettings reads server-settings.json for the given server.
func (m *Manager) LoadServerSettings(name string) (*ServerSettings, error) {
	settings := &ServerSettings{}
	path := filepath.Join(m.ConfigDir(name), "server-settings.json")
	if err := readJSON(path, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// SaveServerSettings writes server-settings.json for the given server.
func (m *Manager) SaveServerSettings(name string, settings *ServerSettings) error {
	path := filepath.Join(m.ConfigDir(name), "server-settings.json")
	return writeJSON(path, settings)
}

// InitServerSettings creates default server-settings.json if it doesn't exist.
func (m *Manager) InitServerSettings(name string) error {
	path := filepath.Join(m.ConfigDir(name), "server-settings.json")
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}
	return m.SaveServerSettings(name, DefaultServerSettings(name))
}

// GetServerSettingValue retrieves a top-level string value from server-settings.json.
func (m *Manager) GetServerSettingValue(serverName, key string) (any, error) {
	path := filepath.Join(m.ConfigDir(serverName), "server-settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	val, ok := raw[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

// SetServerSettingValue sets a top-level key in server-settings.json.
func (m *Manager) SetServerSettingValue(serverName, key string, value any) error {
	path := filepath.Join(m.ConfigDir(serverName), "server-settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	raw[key] = value
	return writeJSON(path, raw)
}
