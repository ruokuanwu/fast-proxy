package caddy

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fast-proxy/internal/config"
)

type Manager struct {
	caddyfile string
	sitesDir  string
}

func NewManager(caddyfile, sitesDir string) *Manager {
	return &Manager{caddyfile: caddyfile, sitesDir: sitesDir}
}

func (m *Manager) Sync(rules []config.Rule) error {
	if err := os.MkdirAll(m.sitesDir, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(m.sitesDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".caddy") {
			continue
		}
		if err := os.Remove(filepath.Join(m.sitesDir, entry.Name())); err != nil {
			return err
		}
	}

	if err := m.writeMainCaddyfile(); err != nil {
		return err
	}

	for _, rule := range rules {
		path := filepath.Join(m.sitesDir, rule.Domain+".caddy")
		content := fmt.Sprintf("%s {\n    reverse_proxy %s\n}\n", rule.Domain, rule.Target)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Reload() error {
	cmd := exec.Command("caddy", "reload", "--config", m.caddyfile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.New("未找到 caddy，请先安装 Caddy")
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("Caddy reload 失败: %s", message)
	}
	return nil
}

func (m *Manager) writeMainCaddyfile() error {
	if err := os.MkdirAll(filepath.Dir(m.caddyfile), 0755); err != nil {
		return err
	}
	content := fmt.Sprintf("import %s/*.caddy\n", m.sitesDir)
	return os.WriteFile(m.caddyfile, []byte(content), 0644)
}
