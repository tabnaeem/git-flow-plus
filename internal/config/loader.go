package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Loader reads and writes Git Flow Plus repository configuration.
//
// Implementations must be safe to construct as zero-value-free instances via
// NewLoader; no package-level state is used so multiple repositories can be
// managed concurrently within a single process.
type Loader interface {
	// Load reads the configuration for the repository rooted at repoRoot.
	// If no config file exists, it returns the default configuration.
	Load(repoRoot string) (*Config, error)
	// Save persists cfg to the repository rooted at repoRoot, creating the
	// Git Flow Plus metadata directory if it does not already exist.
	Save(repoRoot string, cfg *Config) error
	// Exists reports whether a configuration file already exists for the
	// repository rooted at repoRoot.
	Exists(repoRoot string) bool
}

// Path returns the absolute path to the config.json file for the repository
// rooted at repoRoot.
func Path(repoRoot string) string {
	return filepath.Join(repoRoot, DirName, FileName)
}

type viperLoader struct{}

// NewLoader returns a Loader backed by Viper, persisting configuration as
// JSON under <repoRoot>/.gitflowplus/config.json.
func NewLoader() Loader {
	return &viperLoader{}
}

func (l *viperLoader) Exists(repoRoot string) bool {
	_, err := os.Stat(Path(repoRoot))
	return err == nil
}

func (l *viperLoader) Load(repoRoot string) (*Config, error) {
	if !l.Exists(repoRoot) {
		return Default(), nil
	}

	v := viper.New()
	v.SetConfigFile(Path(repoRoot))
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", Path(repoRoot), err)
	}

	cfg := Default()
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", Path(repoRoot), err)
	}

	return cfg, nil
}

func (l *viperLoader) Save(repoRoot string, cfg *Config) error {
	dir := filepath.Join(repoRoot, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: creating %s: %w", dir, err)
	}

	v := viper.New()
	v.SetConfigType("json")
	v.Set("branches", cfg.Branches)
	v.Set("prefixes", cfg.Prefixes)

	target := Path(repoRoot)
	if err := v.WriteConfigAs(target); err != nil {
		return fmt.Errorf("config: writing %s: %w", target, err)
	}

	return nil
}
