package beads

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigFile is where the tracker repository holds its settings: the issue
// ID prefix, one per program, set when tpm init seeds the tracker (srd042).
const ConfigFile = ".beads/config.yaml"

// DefaultPrefix names the issues of a tracker whose config file sets no
// prefix, as every tracker did before the prefix existed.
const DefaultPrefix = "tpm"

// Config is the tracker repository's config file.
type Config struct {
	IssuePrefix string `yaml:"issue-prefix"`
}

var notPrefix = regexp.MustCompile(`[^a-z0-9-]+`)

// PrefixFor derives an issue prefix from a program group: its last path
// segment, lower-cased, keeping letters, digits, and hyphens.
func PrefixFor(group string) string {
	group = strings.TrimSuffix(group, "/")
	p := notPrefix.ReplaceAllString(strings.ToLower(group[strings.LastIndex(group, "/")+1:]), "")
	if p = strings.Trim(p, "-"); p == "" {
		return DefaultPrefix
	}
	return p
}

// WriteConfig writes c as the config file of the tracker clone at dir.
func WriteConfig(dir string, c Config) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ConfigFile), b, 0o644)
}

// prefix is the issue prefix the clone's config file sets, DefaultPrefix
// when it sets none.
func (s *Store) prefix() (string, error) {
	b, err := os.ReadFile(filepath.Join(s.Dir, ConfigFile))
	if errors.Is(err, os.ErrNotExist) {
		return DefaultPrefix, nil
	}
	if err != nil {
		return "", err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return "", fmt.Errorf("%s: %w", ConfigFile, err)
	}
	if c.IssuePrefix == "" {
		return DefaultPrefix, nil
	}
	return c.IssuePrefix, nil
}
