package config_test

import (
	"path/filepath"
	"testing"

	"github.com/tabnaeem/git-flow-plus/internal/config"
)

func TestDefault(t *testing.T) {
	cfg := config.Default()

	if cfg.Branches.Main != "main" {
		t.Errorf("Branches.Main = %q, want %q", cfg.Branches.Main, "main")
	}
	if cfg.Branches.Develop != "develop" {
		t.Errorf("Branches.Develop = %q, want %q", cfg.Branches.Develop, "develop")
	}
	if cfg.Branches.Staging != "staging" {
		t.Errorf("Branches.Staging = %q, want %q", cfg.Branches.Staging, "staging")
	}
	if cfg.Prefixes.ReleaseFix != "release-fix/" {
		t.Errorf("Prefixes.ReleaseFix = %q, want %q", cfg.Prefixes.ReleaseFix, "release-fix/")
	}
	if cfg.Prefixes.DevOps != "release-devops/" {
		t.Errorf("Prefixes.DevOps = %q, want %q", cfg.Prefixes.DevOps, "release-devops/")
	}
}

func TestDefaultLoggingByEnvironment(t *testing.T) {
	cases := map[config.Environment]struct {
		level string
		color bool
	}{
		config.EnvDevelopment:       {"info", true},
		config.EnvTesting:           {"warn", false},
		config.EnvProduction:        {"info", true},
		config.Environment("bogus"): {"info", true}, // unknown falls back to development
	}
	for env, want := range cases {
		got := config.DefaultLogging(env)
		if got.Level != want.level || got.Color != want.color {
			t.Errorf("DefaultLogging(%q) = %+v, want Level=%q Color=%v", env, got, want.level, want.color)
		}
	}
}

func TestDefaultIncludesDevelopmentLogging(t *testing.T) {
	cfg := config.Default()
	if cfg.Environment != config.EnvDevelopment {
		t.Errorf("Environment = %q, want %q", cfg.Environment, config.EnvDevelopment)
	}
	if cfg.Logging.Level != "info" || !cfg.Logging.Color {
		t.Errorf("Logging = %+v, want the development defaults", cfg.Logging)
	}
}

func TestPath(t *testing.T) {
	got := config.Path("/repo")
	want := filepath.Join("/repo", ".gitflowplus", "config.json")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestLoaderExistsFalseForFreshRepo(t *testing.T) {
	dir := t.TempDir()
	loader := config.NewLoader()

	if loader.Exists(dir) {
		t.Fatal("Exists() = true for repo with no config file")
	}
}

func TestLoaderLoadReturnsDefaultWhenMissing(t *testing.T) {
	dir := t.TempDir()
	loader := config.NewLoader()

	cfg, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := config.Default()
	if *cfg != *want {
		t.Errorf("Load() = %+v, want default %+v", cfg, want)
	}
}

func TestLoaderSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	loader := config.NewLoader()

	cfg := config.Default()
	cfg.Branches.Main = "master"
	cfg.Prefixes.Feature = "feat/"

	if err := loader.Save(dir, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !loader.Exists(dir) {
		t.Fatal("Exists() = false after Save()")
	}

	got, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Branches.Main != "master" {
		t.Errorf("Branches.Main = %q, want %q", got.Branches.Main, "master")
	}
	if got.Prefixes.Feature != "feat/" {
		t.Errorf("Prefixes.Feature = %q, want %q", got.Prefixes.Feature, "feat/")
	}
	// Unmodified fields must survive the round trip via defaults + unmarshal.
	if got.Branches.Develop != "develop" {
		t.Errorf("Branches.Develop = %q, want %q", got.Branches.Develop, "develop")
	}
}
