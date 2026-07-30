package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/tabnaeem/git-flow-plus/internal/config"
	"github.com/tabnaeem/git-flow-plus/internal/version"
)

// FileName is the name of the manifest file inside the Git Flow Plus
// metadata directory (config.DirName).
const FileName = "release.json"

// Path returns the absolute path to release.json for the repository rooted
// at repoRoot.
func Path(repoRoot string) string {
	return filepath.Join(repoRoot, config.DirName, FileName)
}

// ArchivePath returns the absolute path to the permanent, historical
// manifest snapshot for a finished release, kept after release.json itself
// is removed from the release branch on finish.
func ArchivePath(repoRoot, release string) string {
	return filepath.Join(repoRoot, config.DirName, "archive", release+".json")
}

// releaseRelPath, versionRelPath, and archiveRelPath return repo-relative
// paths using forward slashes, as required for git command arguments
// (distinct from the OS filesystem paths Path/ArchivePath return above).
func releaseRelPath() string {
	return path.Join(config.DirName, FileName)
}

func versionRelPath() string {
	return path.Join(config.DirName, version.FileName)
}

func archiveRelPath(release string) string {
	return path.Join(config.DirName, "archive", release+".json")
}

// Loader reads and writes release.json.
type Loader interface {
	// Load reads the manifest for the repository rooted at repoRoot.
	Load(repoRoot string) (*Manifest, error)
	// Save persists m to the repository rooted at repoRoot, creating the
	// Git Flow Plus metadata directory if needed.
	Save(repoRoot string, m *Manifest) error
	// Exists reports whether release.json exists for repoRoot.
	Exists(repoRoot string) bool
	// SaveArchive persists a permanent snapshot of m for a finished release
	// under the archive directory, creating it if needed.
	SaveArchive(repoRoot, release string, m *Manifest) error
}

type fileLoader struct{}

// NewLoader returns a Loader backed by plain JSON files.
func NewLoader() Loader {
	return fileLoader{}
}

func (fileLoader) Exists(repoRoot string) bool {
	_, err := os.Stat(Path(repoRoot))
	return err == nil
}

func (fileLoader) Load(repoRoot string) (*Manifest, error) {
	data, err := os.ReadFile(Path(repoRoot))
	if err != nil {
		return nil, fmt.Errorf("release: reading %s: %w", Path(repoRoot), err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("release: parsing %s: %w", Path(repoRoot), err)
	}

	return &m, nil
}

func (fileLoader) Save(repoRoot string, m *Manifest) error {
	dir := filepath.Join(repoRoot, config.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("release: creating %s: %w", dir, err)
	}
	return writeJSON(Path(repoRoot), m)
}

func (fileLoader) SaveArchive(repoRoot, release string, m *Manifest) error {
	dir := filepath.Join(repoRoot, config.DirName, "archive")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("release: creating %s: %w", dir, err)
	}
	return writeJSON(ArchivePath(repoRoot, release), m)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("release: encoding: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("release: writing %s: %w", path, err)
	}
	return nil
}
