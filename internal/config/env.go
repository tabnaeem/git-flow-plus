package config

import (
	"os"
	"strings"
)

// Environment variable names ApplyEnvOverrides recognizes. All are
// optional; an unset or empty variable leaves the corresponding field
// untouched.
const (
	EnvVarEnvironment = "GITFLOWPLUS_ENV"
	EnvVarLogLevel    = "GITFLOWPLUS_LOG_LEVEL"
	EnvVarLogFormat   = "GITFLOWPLUS_LOG_FORMAT"
	EnvVarColor       = "GITFLOWPLUS_COLOR"
	EnvVarVerbose     = "GITFLOWPLUS_VERBOSE"
	EnvVarDebug       = "GITFLOWPLUS_DEBUG"
)

// ApplyEnvOverrides mutates cfg in place, applying any of the
// GITFLOWPLUS_* environment variables that are set. This runs after
// config.json has been loaded (see Loader.Load and cli.App.LoadConfig),
// so the precedence is: built-in defaults < config.json < environment
// variables < explicit CLI flags (flags are applied separately, in
// internal/cli/root.go, since only Cobra knows which flags the user
// actually passed).
//
// GITFLOWPLUS_ENV only reseeds cfg.Logging when the environment variable
// names a different Environment than cfg already has — it never
// overwrites Logging settings a user explicitly saved to config.json for
// their current Environment.
func ApplyEnvOverrides(cfg *Config) {
	if v := Environment(os.Getenv(EnvVarEnvironment)); v != "" && v != cfg.Environment {
		cfg.Environment = v
		cfg.Logging = DefaultLogging(v)
	}
	if v := os.Getenv(EnvVarLogLevel); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv(EnvVarLogFormat); v != "" {
		cfg.Logging.Format = v
	}
	if v, ok := parseBoolEnv(EnvVarColor); ok {
		cfg.Logging.Color = v
	}
	if v, ok := parseBoolEnv(EnvVarVerbose); ok {
		cfg.Logging.Verbose = v
	}
	if v, ok := parseBoolEnv(EnvVarDebug); ok {
		cfg.Logging.Debug = v
	}
}

// parseBoolEnv reads the named environment variable and interprets it as
// a boolean. ok is false when the variable is unset, empty, or not one of
// the recognized spellings, in which case the caller should leave its
// corresponding field unchanged rather than treating it as false.
func parseBoolEnv(name string) (value, ok bool) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch raw {
	case "":
		return false, false
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
