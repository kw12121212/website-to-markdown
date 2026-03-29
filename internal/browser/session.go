package browser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-rod/rod/lib/proto"
)

// sessionCookie is the JSON-serializable representation of a browser cookie.
type sessionCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"http_only"`
}

func sessionPath(site, username string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	return filepath.Join(home, ".wtm", "sessions", fmt.Sprintf("%s-%s.json", site, username)), nil
}

// SaveSession writes browser cookies for a site+account to disk.
func SaveSession(site, username string, cookies []*proto.NetworkCookie) error {
	path, err := sessionPath(site, username)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating session directory: %w", err)
	}
	sc := make([]sessionCookie, len(cookies))
	for i, c := range cookies {
		sc[i] = sessionCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
		}
	}
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadSession reads cookies for a site+account from disk.
// Returns nil, nil if the session file does not exist.
func LoadSession(site, username string) ([]*proto.NetworkCookie, error) {
	path, err := sessionPath(site, username)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}
	var sc []sessionCookie
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}
	cookies := make([]*proto.NetworkCookie, len(sc))
	for i, c := range sc {
		cookies[i] = &proto.NetworkCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
		}
	}
	return cookies, nil
}
