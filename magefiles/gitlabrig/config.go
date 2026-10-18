// Package gitlabrig provisions the GitLab regression rig
// (docs/engineering/eng01-gitlab-regression-rig.yaml): a rig-<run-id>
// subgroup built by the infra/regression Terraform root, read by the
// integration targets, and destroyed afterward. Everything it creates sits
// under the run's subgroup, and teardown removes only that.
package gitlabrig

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/corpus"
)

// ConfigFile is the optional checked-in override at the repository root.
const ConfigFile = "regression.md"

// Config names everything the rig touches.
type Config struct {
	// Group has no default: preflight refuses an empty one by name rather
	// than guessing where to create repositories.
	Group           string        `yaml:"group"`
	BaseURL         string        `yaml:"base_url"`
	POCount         int           `yaml:"po_repos"`
	Reports         *bool         `yaml:"reports"`
	SweepAge        time.Duration `yaml:"sweep_age"`
	ServiceAccounts bool          `yaml:"service_accounts"`
}

// Defaults are the literal settings an absent regression.md means.
func Defaults() Config {
	reports := true
	return Config{BaseURL: "https://gitlab.com", POCount: 2, Reports: &reports, SweepAge: 24 * time.Hour}
}

// LoadConfig reads regression.md, a config document (design D11), under
// root over the defaults. An absent
// file is the defaults; a malformed one is a named error.
func LoadConfig(root string) (Config, error) {
	cfg := Defaults()
	path := filepath.Join(root, ConfigFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := corpus.DecodeConfig(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s is malformed: %w", path, err)
	}
	if cfg.Reports == nil {
		cfg.Reports = Defaults().Reports
	}
	return cfg, nil
}

// Validate refuses settings the rig cannot act on, naming each one.
func (c Config) Validate() error {
	if c.Group == "" {
		return fmt.Errorf("%s names no group; set group to the path of the group rig subgroups go under", ConfigFile)
	}
	if c.POCount < 1 {
		return fmt.Errorf("%s: po_repos must be at least 1, got %d", ConfigFile, c.POCount)
	}
	if u, err := url.Parse(c.BaseURL); err != nil || u.Host == "" {
		return fmt.Errorf("%s: base_url %q is not a URL", ConfigFile, c.BaseURL)
	}
	return nil
}

// Host is the instance host name, as glab names it.
func (c Config) Host() string {
	u, _ := url.Parse(c.BaseURL)
	return u.Host
}

// APIURL is the REST base for the instance.
func (c Config) APIURL() string { return c.BaseURL + "/api/v4" }
