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

func (m *Manager) Init() error {
	if err := requireCaddy(); err != nil {
		return err
	}
	if err := os.MkdirAll(m.sitesDir, 0755); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("无权限创建 %s，请使用 sudo 重新执行", m.sitesDir)
		}
		return err
	}
	if err := m.ensureImport(); err != nil {
		return err
	}
	if err := m.Validate(); err != nil {
		return err
	}
	return nil
}

func (m *Manager) EnsureInitialized() error {
	data, err := os.ReadFile(m.caddyfile)
	if err != nil {
		return err
	}
	if !hasImportLine(string(data), m.importLine()) {
		return fmt.Errorf("未检测到 fast-proxy Caddy import，请先执行: sudo fast-proxy init")
	}
	return nil
}

func (m *Manager) Sync(rules []config.Rule) error {
	if err := m.EnsureInitialized(); err != nil {
		return err
	}
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

	for _, rule := range rules {
		path := filepath.Join(m.sitesDir, rule.Domain+".caddy")
		content := fmt.Sprintf("%s {\n    tls internal\n    reverse_proxy %s\n}\n", rule.Domain, rule.Target)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Reload() error {
	if err := m.EnsureInitialized(); err != nil {
		return err
	}
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

func (m *Manager) Validate() error {
	cmd := exec.Command("caddy", "validate", "--config", m.caddyfile)
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
		return fmt.Errorf("Caddy validate 失败: %s", message)
	}
	return nil
}

func (m *Manager) ensureImport() error {
	if err := os.MkdirAll(filepath.Dir(m.caddyfile), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(m.caddyfile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	content := string(data)
	line := m.importLine()
	if hasImportLine(content, line) {
		return nil
	}
	if strings.TrimSpace(content) != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n# fast-proxy\n" + line + "\n"
	if err := os.WriteFile(m.caddyfile, []byte(content), 0644); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("无权限修改 %s，请使用 sudo 重新执行", m.caddyfile)
		}
		return err
	}
	return nil
}

func (m *Manager) importLine() string {
	return fmt.Sprintf("import %s/*.caddy", m.sitesDir)
}

func hasImportLine(content, line string) bool {
	for _, existing := range strings.Split(content, "\n") {
		if strings.TrimSpace(existing) == line {
			return true
		}
	}
	return false
}

func requireCaddy() error {
	if _, err := exec.LookPath("caddy"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.New("未找到 caddy，请先安装 Caddy")
		}
		return err
	}
	return nil
}
