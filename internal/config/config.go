// Package config loads and persists Git Flow Plus repository configuration
// from .gitflowplus/config.json.
package config

// DirName is the name of the Git Flow Plus metadata directory created inside
// an initialized repository.
const DirName = ".gitflowplus"

// FileName is the name of the configuration file inside DirName.
const FileName = "config.json"

// Branches names the long-lived branches Git Flow Plus operates against.
type Branches struct {
	Main    string `json:"main" mapstructure:"main"`
	Develop string `json:"develop" mapstructure:"develop"`
	Staging string `json:"staging" mapstructure:"staging"`
}

// Prefixes names the branch prefixes used for each short-lived workflow
// type. There is no "release" prefix: releases live on the permanent
// staging branch (config.Branches.Staging), not on an ephemeral
// release/<name> branch.
type Prefixes struct {
	Feature    string `json:"feature" mapstructure:"feature"`
	Hotfix     string `json:"hotfix" mapstructure:"hotfix"`
	Support    string `json:"support" mapstructure:"support"`
	ReleaseFix string `json:"releaseFix" mapstructure:"releaseFix"`
	DevOps     string `json:"devops" mapstructure:"devops"`
	VersionTag string `json:"versionTag" mapstructure:"versionTag"`
}

// Environment selects which built-in Logging defaults Default() applies —
// development favors maximum visibility, testing favors quiet/deterministic
// output, production favors the calm, colorized default a release manager
// would want. It has no effect beyond seeding those defaults: nothing else
// in Git Flow Plus branches on it, and a saved config.json's Logging block
// always wins over re-deriving it from Environment.
type Environment string

const (
	// EnvDevelopment is the default: verbose enough to see what's
	// happening while working on a repository day to day.
	EnvDevelopment Environment = "development"
	// EnvTesting favors quiet, uncolored, deterministic output — the
	// shape you want in a test harness or CI log.
	EnvTesting Environment = "testing"
	// EnvProduction favors calm, human-readable, colorized output.
	EnvProduction Environment = "production"
)

// Logging configures Git Flow Plus's own diagnostic logging (see
// internal/logging). Values here are defaults only — the root command's
// --verbose/--debug/--json-log/--no-color flags, when explicitly passed,
// always take precedence over whatever is configured here.
type Logging struct {
	// Level is one of "trace", "debug", "info", "success", "warn",
	// "error", "fatal" (see logging.ParseLevel; unrecognized values fall
	// back to "info").
	Level string `json:"level" mapstructure:"level"`
	// Format is "text" (human-readable, optionally colorized) or "json"
	// (newline-delimited JSON, for CI/CD log aggregation).
	Format string `json:"format" mapstructure:"format"`
	// Color enables ANSI color codes in text-format output. Still
	// subject to runtime detection — no effect on a non-terminal
	// destination or when NO_COLOR is set.
	Color bool `json:"color" mapstructure:"color"`
	// Verbose enables debug-level logging (shorthand for Level="debug").
	Verbose bool `json:"verbose" mapstructure:"verbose"`
	// Debug enables trace-level logging, the most detailed level
	// (shorthand for Level="trace"). Also the flag that would reveal
	// stack traces, if any of Git Flow Plus's errors carried one.
	Debug bool `json:"debug" mapstructure:"debug"`
}

// Config is the persisted Git Flow Plus repository configuration.
type Config struct {
	Environment Environment `json:"environment" mapstructure:"environment"`
	Branches    Branches    `json:"branches" mapstructure:"branches"`
	Prefixes    Prefixes    `json:"prefixes" mapstructure:"prefixes"`
	Logging     Logging     `json:"logging" mapstructure:"logging"`
}

// DefaultLogging returns the built-in Logging defaults for env. Unknown
// Environment values are treated as EnvDevelopment.
func DefaultLogging(env Environment) Logging {
	switch env {
	case EnvTesting:
		return Logging{Level: "warn", Format: "text", Color: false}
	case EnvProduction:
		return Logging{Level: "info", Format: "text", Color: true}
	default: // EnvDevelopment and anything unrecognized
		return Logging{Level: "info", Format: "text", Color: true}
	}
}

// Default returns the built-in default configuration, matching the
// conventions of the original Git Flow tooling with Git Flow Plus additions.
func Default() *Config {
	return &Config{
		Environment: EnvDevelopment,
		Branches: Branches{
			Main:    "main",
			Develop: "develop",
			Staging: "staging",
		},
		Prefixes: Prefixes{
			Feature:    "feature/",
			Hotfix:     "hotfix/",
			Support:    "support/",
			ReleaseFix: "release-fix/",
			DevOps:     "release-devops/",
			VersionTag: "v",
		},
		Logging: DefaultLogging(EnvDevelopment),
	}
}
