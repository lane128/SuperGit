package desktop

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type preferences struct {
	Dark     bool
	FontSize float32
	Recent   []string
}

func (p *preferences) remember(path string) {
	list := []string{path}
	for _, old := range p.Recent {
		if old != path && len(list) < 12 {
			list = append(list, old)
		}
	}
	p.Recent = list
}
func loadPreferences(path string) (preferences, error) {
	p := preferences{FontSize: 13}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal(b, &p); err != nil {
		return preferences{FontSize: 13}, err
	}
	if p.FontSize < 11 || p.FontSize > 19 {
		p.FontSize = 13
	}
	if len(p.Recent) > 12 {
		p.Recent = p.Recent[:12]
	}
	return p, nil
}
func savePreferences(path string, p preferences) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".preferences-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func (a *app) persist() {
	a.prefs.Dark, a.prefs.FontSize = a.dark, a.fontSize
	if err := savePreferences(a.prefsPath, a.prefs); err != nil {
		a.record("保存偏好失败", err)
	}
}
func displayURL(address string) string {
	u, err := url.Parse(address)
	if err == nil && u.Scheme != "" {
		u.User = nil
		u.RawQuery = ""
		u.Fragment = ""
		return u.String()
	}
	// SCP-style SSH paths may contain a username, but never display passwords.
	if at := strings.Index(address, "@"); at >= 0 {
		return "…" + address[at:]
	}
	return address
}
