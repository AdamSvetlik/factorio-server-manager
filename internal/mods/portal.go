package mods

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const (
	modPortalBase = "https://mods.factorio.com"
	apiBase       = modPortalBase + "/api/mods"
)

// Client interacts with the Factorio mod portal API.
type Client struct {
	http     *http.Client
	username string
	token    string
}

// NewClient creates a new mod portal client.
func NewClient(username, token string) *Client {
	return &Client{
		http:     &http.Client{Timeout: 30 * time.Second},
		username: username,
		token:    token,
	}
}

// ModRelease describes a single release of a mod.
type ModRelease struct {
	DownloadURL string    `json:"download_url"`
	FileName    string    `json:"file_name"`
	Version     string    `json:"version"`
	ReleasedAt  time.Time `json:"released_at"`
	SHA1        string    `json:"sha1"`
	InfoJSON    struct {
		FactorioVersion string `json:"factorio_version"`
	} `json:"info_json"`
}

// ModInfo holds metadata for a mod.
type ModInfo struct {
	Name          string       `json:"name"`
	Title         string       `json:"title"`
	Owner         string       `json:"owner"`
	Summary       string       `json:"summary"`
	DownloadsCount int         `json:"downloads_count"`
	Category      string       `json:"category"`
	Score         float64      `json:"score"`
	LatestRelease *ModRelease  `json:"latest_release"`
	Releases      []ModRelease `json:"releases,omitempty"`
}

// SearchResult wraps a paginated search response.
type SearchResult struct {
	Pagination struct {
		Count     int `json:"count"`
		Page      int `json:"page"`
		PageCount int `json:"page_count"`
		PageSize  int `json:"page_size"`
	} `json:"pagination"`
	Results []ModInfo `json:"results"`
}

// Search queries the mod portal for mods matching the query string.
func (c *Client) Search(query string, page, pageSize int) (*SearchResult, error) {
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("page_size", fmt.Sprintf("%d", pageSize))
	if query != "" {
		params.Set("q", query)
	}

	resp, err := c.http.Get(apiBase + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("mod search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mod portal returned %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return &result, nil
}

// GetMod fetches full info (including all releases) for a mod by name.
func (c *Client) GetMod(name string) (*ModInfo, error) {
	resp, err := c.http.Get(apiBase + "/" + url.PathEscape(name) + "/full")
	if err != nil {
		return nil, fmt.Errorf("mod info request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("mod %q not found", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mod portal returned %d", resp.StatusCode)
	}

	var info ModInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode mod info: %w", err)
	}
	return &info, nil
}

// Download downloads a specific mod release into the destination directory.
// Requires valid username and token.
func (c *Client) Download(downloadURL, destDir, fileName string) error {
	if c.username == "" || c.token == "" {
		return fmt.Errorf("downloading mods requires Factorio credentials — run: factorio-server-manager auth login")
	}

	params := url.Values{}
	params.Set("username", c.username)
	params.Set("token", c.token)
	fullURL := modPortalBase + downloadURL + "?" + params.Encode()

	resp, err := c.http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("download mod: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	destPath := filepath.Join(destDir, fileName)
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create mod file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(destPath) //nolint:errcheck
		return fmt.Errorf("write mod file: %w", err)
	}
	return nil
}

// ListInstalled returns the names of .zip mod files in the given directory.
func ListInstalled(modsDir string) ([]string, error) {
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return nil, fmt.Errorf("read mods dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".zip" {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// RemoveMod deletes a mod .zip file from the mods directory.
func RemoveMod(modsDir, fileName string) error {
	path := filepath.Join(modsDir, fileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("mod file %q not found in mods directory", fileName)
	}
	return os.Remove(path)
}
