package version

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DirName is the Git Flow Plus metadata directory (shared with
// internal/config and internal/release).
const DirName = ".gitflowplus"

// FileName is the name of the version file inside DirName.
const FileName = "version.json"

// Path returns the absolute path to version.json for the repository rooted
// at repoRoot.
func Path(repoRoot string) string {
	return filepath.Join(repoRoot, DirName, FileName)
}

// Loader reads and writes version.json.
type Loader interface {
	// Load reads the version for the repository rooted at repoRoot.
	Load(repoRoot string) (*Version, error)
	// Save persists v to the repository rooted at repoRoot, creating the
	// Git Flow Plus metadata directory if needed.
	Save(repoRoot string, v *Version) error
	// Exists reports whether version.json exists for repoRoot.
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

func (fileLoader) Load(repoRoot string) (*Version, error) {
	data, err := os.ReadFile(Path(repoRoot))
	if err != nil {
		return nil, fmt.Errorf("version: reading %s: %w", Path(repoRoot), err)
	}

	var v Version
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("version: parsing %s: %w", Path(repoRoot), err)
	}

	return &v, nil
}

func (fileLoader) Save(repoRoot string, v *Version) error {
	dir := filepath.Join(repoRoot, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("version: creating %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("version: encoding: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(Path(repoRoot), data, 0o644); err != nil {
		return fmt.Errorf("version: writing %s: %w", Path(repoRoot), err)
	}

	return nil
}
