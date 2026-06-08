package hosts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fast-proxy/internal/config"
)

const marker = "# fast-proxy"

type File struct {
	path string
}

func NewFile(path string) *File {
	return &File{path: path}
}

func (f *File) Sync(rules []config.Rule) error {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	next := make([]string, 0, len(lines)+len(rules))
	for _, line := range lines {
		if strings.Contains(line, marker) {
			continue
		}
		next = append(next, line)
	}

	for _, rule := range rules {
		next = append(next, fmt.Sprintf("127.0.0.1 %s %s", rule.Domain, marker))
	}

	content := strings.TrimRight(strings.Join(next, "\n"), "\n") + "\n"
	if err := writeFileAtomic(f.path, []byte(content), 0644); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("无权限修改 %s，请使用 sudo 重新执行", f.path)
		}
		return err
	}
	return nil
}

func writeFileAtomic(path string, data []byte, fallbackMode os.FileMode) error {
	mode := fallbackMode
	if stat, err := os.Stat(path); err == nil {
		mode = stat.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".fast-proxy-hosts-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
