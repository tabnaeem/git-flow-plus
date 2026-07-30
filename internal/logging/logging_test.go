package logging_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tabnaeem/git-flow-plus/internal/logging"
)

func TestNewTextFormatDefaultLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText})

	logger.Debug("hidden")
	if buf.Len() != 0 {
		t.Fatalf("debug message logged at default (info) level: %q", buf.String())
	}

	logger.Info("visible", "key", "value")
	if !strings.Contains(buf.String(), "visible") {
		t.Errorf("output = %q, want it to contain %q", buf.String(), "visible")
	}
}

func TestNewVerboseEnablesDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Verbose: true, Format: logging.FormatText})

	logger.Debug("shown")
	if !strings.Contains(buf.String(), "shown") {
		t.Errorf("output = %q, want debug message present when Verbose=true", buf.String())
	}
}

func TestNewJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatJSON})

	logger.Info("payload", "cmd", "release-status")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("output is not valid JSON: %v (%q)", err, buf.String())
	}
	if record["msg"] != "payload" {
		t.Errorf("record[msg] = %v, want %q", record["msg"], "payload")
	}
	if record["cmd"] != "release-status" {
		t.Errorf("record[cmd] = %v, want %q", record["cmd"], "release-status")
	}
}

func TestNewReturnsUsableLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf})

	if logger == nil {
		t.Fatal("New() returned nil")
	}
}

func TestConsoleFormatRendersBracketedLevelAndAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText})

	logger.Info("Current Branch", "branch", "staging")

	got := buf.String()
	want := "[INFO] Current Branch  branch: staging\n"
	if got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestConsoleFormatWithoutColorHasNoANSICodes(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText, Color: false})

	logger.Warn("careful")

	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("output = %q, want no ANSI escape codes when Color is false", buf.String())
	}
}

func TestConsoleFormatWithColorAddsANSICodes(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText, Color: true})

	logger.Warn("careful")

	if !strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("output = %q, want ANSI escape codes when Color is true", buf.String())
	}
	if !strings.Contains(buf.String(), "[WARN]") {
		t.Errorf("output = %q, want it to still contain the plain [WARN] label text", buf.String())
	}
}

func TestTraceHiddenUnlessTraceOption(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText, Verbose: true})

	logging.Trace(logger, "very detailed")
	if buf.Len() != 0 {
		t.Errorf("Trace message shown with only Verbose set (Debug level): %q", buf.String())
	}

	logger = logging.New(logging.Options{Writer: &buf, Format: logging.FormatText, Trace: true})
	logging.Trace(logger, "very detailed")
	if !strings.Contains(buf.String(), "[TRACE] very detailed") {
		t.Errorf("output = %q, want the trace message shown with Trace option set", buf.String())
	}
}

func TestTraceOptionImpliesVerbose(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText, Trace: true})

	logger.Debug("debug detail")
	if !strings.Contains(buf.String(), "[DEBUG] debug detail") {
		t.Errorf("output = %q, want debug messages shown when Trace is set", buf.String())
	}
}

func TestSuccessRendersDistinctLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText})

	logging.Success(logger, "Release Build completed successfully.")
	if !strings.Contains(buf.String(), "[SUCCESS] Release Build completed successfully.") {
		t.Errorf("output = %q, want a [SUCCESS] line", buf.String())
	}
}

func TestSuccessShownAtDefaultLevel(t *testing.T) {
	var buf bytes.Buffer
	// Default level is Info; Success (between Info and Warn) must still
	// pass the threshold without needing Verbose.
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText})

	logging.Success(logger, "done")
	if buf.Len() == 0 {
		t.Error("Success message hidden at default level, want it shown (Success > Info)")
	}
}

func TestJSONFormatUsesRealLevelNames(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatJSON})

	logger.Warn("careful")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("output is not valid JSON: %v (%q)", err, buf.String())
	}
	if record["level"] != "WARN" {
		t.Errorf("record[level] = %v, want %q", record["level"], "WARN")
	}
}

func TestExplicitLevelOverridesVerbose(t *testing.T) {
	var buf bytes.Buffer
	warn := logging.LevelWarn
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText, Verbose: true, Level: &warn})

	logger.Info("suppressed by explicit Level despite Verbose")
	if buf.Len() != 0 {
		t.Errorf("output = %q, want Info suppressed when Level is explicitly Warn", buf.String())
	}

	logger.Warn("shown")
	if !strings.Contains(buf.String(), "shown") {
		t.Errorf("output = %q, want the Warn message shown", buf.String())
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]int{
		"trace":   -8,
		"debug":   -4,
		"info":    0,
		"success": 2,
		"warn":    4,
		"error":   8,
		"fatal":   12,
		"unknown": 0, // falls back to Info
	}
	for name, want := range cases {
		if got := int(logging.ParseLevel(name)); got != want {
			t.Errorf("ParseLevel(%q) = %d, want %d", name, got, want)
		}
	}
}
