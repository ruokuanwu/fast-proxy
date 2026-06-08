package config

import (
	"os"
	"os/user"
	"path/filepath"
)

type Paths struct {
	HomeDir    string
	ConfigDir  string
	ConfigFile string
	Caddyfile  string
	SitesDir   string
}

const (
	DefaultCaddyfile = "/etc/caddy/Caddyfile"
	DefaultSitesDir  = "/etc/caddy/fast-proxy"
)

func ResolvePaths() (Paths, error) {
	home, err := resolveHomeDir()
	if err != nil {
		return Paths{}, err
	}

	configDir := filepath.Join(home, ".fast-proxy")
	return Paths{
		HomeDir:    home,
		ConfigDir:  configDir,
		ConfigFile: filepath.Join(configDir, "config.json"),
		Caddyfile:  DefaultCaddyfile,
		SitesDir:   DefaultSitesDir,
	}, nil
}

func resolveHomeDir() (string, error) {
	if home := os.Getenv("FAST_PROXY_HOME"); home != "" {
		return home, nil
	}
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" && sudoUser != "root" {
		u, err := user.Lookup(sudoUser)
		if err == nil && u.HomeDir != "" {
			return u.HomeDir, nil
		}
	}
	return os.UserHomeDir()
}
