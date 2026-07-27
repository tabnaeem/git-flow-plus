package hooks_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/hulhub/git-flow-plus/internal/hooks"
)

// writeHook writes a hook script appropriate for the current OS (.bat on
// Windows, a shebang .sh script with the execute bit set elsewhere) that
// writes the value of env var MARKER_VAR to marker.txt in repoPath, then
// exits with exitCode.
func writeHook(t *testing.T, repoPath, name string, exitCode int) {
	t.Helper()
	dir := filepath.Join(repoPath, ".gitflowplus", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if runtime.GOOS == "windows" {
		script := "@echo off\r\n" +
			"echo %MARKER_VAR%> \"" + filepath.Join(repoPath, "marker.txt") + "\"\r\n" +
			"exit /b " + strconv.Itoa(exitCode) + "\r\n"
		path := filepath.Join(dir, name+".bat")
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return
	}

	script := "#!/bin/sh\n" +
		"echo \"$MARKER_VAR\" > \"" + filepath.Join(repoPath, "marker.txt") + "\"\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	path := filepath.Join(dir, name+".sh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestRunReturnsFalseWhenNoHookExists(t *testing.T) {
	dir := t.TempDir()
	ran, err := hooks.NewRunner().Run(context.Background(), dir, "post-qa-tag", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if ran {
		t.Error("Run() ran = true, want false when no hook script exists")
	}
}

func TestRunExecutesHookAndPassesEnv(t *testing.T) {
	dir := t.TempDir()
	writeHook(t, dir, "post-qa-tag", 0)

	ran, err := hooks.NewRunner().Run(context.Background(), dir, "post-qa-tag", map[string]string{
		"MARKER_VAR": "hello-from-gitflowplus",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ran {
		t.Fatal("Run() ran = false, want true")
	}

	data, err := os.ReadFile(filepath.Join(dir, "marker.txt"))
	if err != nil {
		t.Fatalf("reading marker.txt: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != "hello-from-gitflowplus" {
		t.Errorf("marker.txt content = %q, want %q", got, "hello-from-gitflowplus")
	}
}

func TestRunReportsScriptFailure(t *testing.T) {
	dir := t.TempDir()
	writeHook(t, dir, "post-production-tag", 1)

	ran, err := hooks.NewRunner().Run(context.Background(), dir, "post-production-tag", nil)
	if !ran {
		t.Error("Run() ran = false, want true even though the script failed")
	}
	if err == nil {
		t.Fatal("Run() error = nil, want failure for a non-zero exit code")
	}
}

func TestRunIgnoresUnrelatedHookNames(t *testing.T) {
	dir := t.TempDir()
	writeHook(t, dir, "post-qa-tag", 0)

	ran, err := hooks.NewRunner().Run(context.Background(), dir, "post-production-tag", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if ran {
		t.Error("Run() ran = true for a hook name with no matching script")
	}
}
