package logging_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hulhub/git-flow-plus/internal/logging"
)

func TestLogErrorPlainErrorHasNoExtraLines(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText})

	logging.LogError(logger, errors.New("something broke"))

	got := buf.String()
	if !strings.Contains(got, "[ERROR] something broke") {
		t.Errorf("output = %q, want it to contain the plain error message", got)
	}
	if strings.Contains(got, "Cause:") || strings.Contains(got, "Resolution:") {
		t.Errorf("output = %q, want no Cause/Resolution lines for a plain error", got)
	}
}

func TestLogErrorAppErrorIncludesCauseAndResolution(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText})

	err := logging.NewAppError(errors.New("config: reading .gitflowplus/config.json: invalid character"),
		"the configuration file contains malformed JSON",
		"fix or delete .gitflowplus/config.json, or run 'git flow init' to regenerate defaults")

	logging.LogError(logger, err)

	got := buf.String()
	for _, want := range []string{
		"[ERROR] config: reading",
		"Cause: the configuration file contains malformed JSON",
		"Resolution: fix or delete .gitflowplus/config.json",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want it to contain %q", got, want)
		}
	}
}

func TestLogErrorFindsCauseThroughWrapChain(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{Writer: &buf, Format: logging.FormatText})

	inner := logging.NewAppError(errors.New("root cause"), "disk full", "free up space")
	outer := fmt.Errorf("saving manifest: %w", inner)

	logging.LogError(logger, outer)

	got := buf.String()
	if !strings.Contains(got, "Cause: disk full") {
		t.Errorf("output = %q, want the wrapped AppError's Cause surfaced", got)
	}
	if !strings.Contains(got, "Resolution: free up space") {
		t.Errorf("output = %q, want the wrapped AppError's Resolution surfaced", got)
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	sentinel := errors.New("sentinel")
	wrapped := logging.NewAppError(sentinel, "cause", "resolution")

	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is(wrapped, sentinel) = false, want true (Unwrap must expose the original error)")
	}
}
