package avd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config is a config.ini that preserves comments, key order and line endings
// when a value changes. AVD config files are small, so holding every line is
// simpler and safer than rewriting the whole file from a map.
type Config struct {
	path            string
	lines           []string
	lineEnding      string
	trailingNewline bool
}

// LoadConfig reads an AVD config.ini.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text := string(data)
	lineEnding := "\n"
	if strings.Contains(text, "\r\n") {
		lineEnding = "\r\n"
	}
	trailingNewline := strings.HasSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\n")

	var lines []string
	if text != "" {
		lines = strings.Split(text, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSuffix(line, "\r")
		}
	}

	return &Config{
		path:            path,
		lines:           lines,
		lineEnding:      lineEnding,
		trailingNewline: trailingNewline,
	}, nil
}

// Get returns the first value for key.
func (c *Config) Get(key string) (string, bool) {
	for _, line := range c.lines {
		if k, v, ok := splitKeyValue(line); ok && k == key {
			return v, true
		}
	}
	return "", false
}

// Set updates every occurrence of key, dropping duplicates, and appends the
// key when it is missing. It returns the previous value and whether the key
// already existed.
func (c *Config) Set(key, value string) (old string, existed bool) {
	out := make([]string, 0, len(c.lines)+1)
	written := false

	for _, line := range c.lines {
		k, v, ok := splitKeyValue(line)
		if !ok || k != key {
			out = append(out, line)
			continue
		}
		if written {
			continue
		}
		old, existed, written = v, true, true
		out = append(out, key+"="+value)
	}

	if !written {
		out = append(out, key+"="+value)
	}
	c.lines = out
	return old, existed
}

// Delete removes every occurrence of key and returns the previous value.
func (c *Config) Delete(key string) (old string, existed bool) {
	out := make([]string, 0, len(c.lines))
	for _, line := range c.lines {
		k, v, ok := splitKeyValue(line)
		if ok && k == key {
			if !existed {
				old = v
			}
			existed = true
			continue
		}
		out = append(out, line)
	}
	c.lines = out
	return old, existed
}

// Bytes renders the file exactly as Save would write it.
func (c *Config) Bytes() []byte {
	text := strings.Join(c.lines, c.lineEnding)
	if c.trailingNewline && text != "" {
		text += c.lineEnding
	}
	return []byte(text)
}

// Save writes the config atomically, keeping the original permissions.
func (c *Config) Save() error {
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(c.path); err == nil {
		mode = info.Mode().Perm()
	}
	return writeFileAtomic(c.path, c.Bytes(), mode)
}

func splitKeyValue(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
		return "", "", false
	}
	separator := strings.Index(trimmed, "=")
	if separator <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(trimmed[:separator]), strings.TrimSpace(trimmed[separator+1:]), true
}

func writeFileAtomic(path string, data []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gadb-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

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

	if err := os.Rename(tmpName, path); err == nil {
		return nil
	}
	// On Windows os.Rename cannot replace an existing file.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpName, path)
}
