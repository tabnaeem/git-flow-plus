package config_test

import (
	"testing"

	"github.com/tabnaeem/git-flow-plus/internal/config"
)

func TestApplyEnvOverridesLeavesConfigUntouchedWhenUnset(t *testing.T) {
	cfg := config.Default()
	want := *cfg

	config.ApplyEnvOverrides(cfg)

	if *cfg != want {
		t.Errorf("ApplyEnvOverrides() with no env vars set changed config: got %+v, want %+v", cfg, want)
	}
}

func TestApplyEnvOverridesLogLevel(t *testing.T) {
	t.Setenv(config.EnvVarLogLevel, "trace")
	cfg := config.Default()

	config.ApplyEnvOverrides(cfg)

	if cfg.Logging.Level != "trace" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "trace")
	}
}

func TestApplyEnvOverridesLogFormat(t *testing.T) {
	t.Setenv(config.EnvVarLogFormat, "json")
	cfg := config.Default()

	config.ApplyEnvOverrides(cfg)

	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q, want %q", cfg.Logging.Format, "json")
	}
}

func TestApplyEnvOverridesBooleans(t *testing.T) {
	t.Setenv(config.EnvVarColor, "false")
	t.Setenv(config.EnvVarVerbose, "true")
	t.Setenv(config.EnvVarDebug, "1")
	cfg := config.Default()

	config.ApplyEnvOverrides(cfg)

	if cfg.Logging.Color {
		t.Error("Logging.Color = true, want false (GITFLOWPLUS_COLOR=false)")
	}
	if !cfg.Logging.Verbose {
		t.Error("Logging.Verbose = false, want true (GITFLOWPLUS_VERBOSE=true)")
	}
	if !cfg.Logging.Debug {
		t.Error("Logging.Debug = false, want true (GITFLOWPLUS_DEBUG=1)")
	}
}

func TestApplyEnvOverridesUnrecognizedBooleanLeavesFieldUnchanged(t *testing.T) {
	t.Setenv(config.EnvVarColor, "maybe")
	cfg := config.Default()
	want := cfg.Logging.Color

	config.ApplyEnvOverrides(cfg)

	if cfg.Logging.Color != want {
		t.Errorf("Logging.Color = %v, want unchanged %v for an unrecognized value", cfg.Logging.Color, want)
	}
}

func TestApplyEnvOverridesEnvironmentReseedsLoggingDefaults(t *testing.T) {
	t.Setenv(config.EnvVarEnvironment, "testing")
	cfg := config.Default() // starts as EnvDevelopment

	config.ApplyEnvOverrides(cfg)

	if cfg.Environment != config.EnvTesting {
		t.Errorf("Environment = %q, want %q", cfg.Environment, config.EnvTesting)
	}
	if cfg.Logging.Level != "warn" || cfg.Logging.Color {
		t.Errorf("Logging = %+v, want the testing-environment defaults (warn, no color)", cfg.Logging)
	}
}

func TestApplyEnvOverridesEnvironmentDoesNotOverwriteExplicitLoggingWhenUnchanged(t *testing.T) {
	t.Setenv(config.EnvVarEnvironment, "development")
	cfg := config.Default() // already EnvDevelopment
	cfg.Logging.Level = "error"

	config.ApplyEnvOverrides(cfg)

	if cfg.Logging.Level != "error" {
		t.Errorf("Logging.Level = %q, want unchanged %q (GITFLOWPLUS_ENV matches the existing Environment)", cfg.Logging.Level, "error")
	}
}
