package feature

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tabnaeem/git-flow-plus/internal/config"
)

// FileName is the name of the registry file inside the Git Flow Plus
// metadata directory (config.DirName).
const FileName = "features.json"

// Path returns the absolute path to features.json for the repository
// rooted at repoRoot.
func Path(repoRoot string) string {
	return filepath.Join(repoRoot, config.DirName, FileName)
}

// RelPath returns the repo-relative path (forward slashes) to
// features.json, as required for git command arguments (git wants "/"
// even on Windows, unlike Path's OS filesystem path).
func RelPath() string {
	return config.DirName + "/" + FileName
}

// Loader reads and writes features.json.
type Loader interface {
	// Load reads the registry for the repository rooted at repoRoot,
	// returning an empty Registry if none exists yet.
	Load(repoRoot string) (*Registry, error)
	// Save persists r to the repository rooted at repoRoot, creating the
	// Git Flow Plus metadata directory if needed.
	Save(repoRoot string, r *Registry) error
	// Exists reports whether features.json exists for repoRoot.
	Exists(repoRoot string) bool
}

type fileLoader struct{}

// NewLoader returns a Loader backed by a plain JSON file.
func NewLoader() Loader {
	return fileLoader{}
}

func (fileLoader) Exists(repoRoot string) bool {
	_, err := os.Stat(Path(repoRoot))
	return err == nil
}

func (l fileLoader) Load(repoRoot string) (*Registry, error) {
	if !l.Exists(repoRoot) {
		return New(), nil
	}

	data, err := os.ReadFile(Path(repoRoot))
	if err != nil {
		return nil, fmt.Errorf("feature: reading %s: %w", Path(repoRoot), err)
	}

	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("feature: parsing %s: %w", Path(repoRoot), err)
	}
	if r.Features == nil {
		r.Features = []Feature{}
	}

	return &r, nil
}

func (fileLoader) Save(repoRoot string, r *Registry) error {
	dir := filepath.Join(repoRoot, config.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("feature: creating %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("feature: encoding: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(Path(repoRoot), data, 0o644); err != nil {
		return fmt.Errorf("feature: writing %s: %w", Path(repoRoot), err)
	}
	return nil
}
