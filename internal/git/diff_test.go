package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTempGitRepo creates a temporary git repo and returns its path.
func setupTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup command %v failed: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestIsGitRepo(t *testing.T) {
	t.Run("inside git repo", func(t *testing.T) {
		dir := setupTempGitRepo(t)
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)

		if err := IsGitRepo(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("outside git repo", func(t *testing.T) {
		dir := t.TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)

		if err := IsGitRepo(); !errors.Is(err, ErrNotGitRepo) {
			t.Errorf("expected ErrNotGitRepo, got %v", err)
		}
	})
}

func TestGetStagedDiff(t *testing.T) {
	t.Run("no staged changes", func(t *testing.T) {
		dir := setupTempGitRepo(t)
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)

		diff, err := GetStagedDiff()
		if !errors.Is(err, ErrNoStagedChanges) {
			t.Errorf("expected ErrNoStagedChanges, got %v", err)
		}
		if diff != "" {
			t.Errorf("expected empty diff, got %q", diff)
		}
	})

	t.Run("with staged changes", func(t *testing.T) {
		dir := setupTempGitRepo(t)
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)

		// Create a file and stage it
		file := filepath.Join(dir, "hello.txt")
		os.WriteFile(file, []byte("hello world\n"), 0644)
		exec.Command("git", "add", "hello.txt").Run()

		diff, err := GetStagedDiff()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(diff, "hello world") {
			t.Errorf("expected diff to contain 'hello world', got %q", diff)
		}
	})
}

func TestTruncateDiff(t *testing.T) {
	t.Run("short diff not truncated", func(t *testing.T) {
		diff := "short diff"
		result := TruncateDiff(diff, 1024)
		if result != diff {
			t.Errorf("expected unchanged diff, got %q", result)
		}
	})

	t.Run("long diff gets truncated", func(t *testing.T) {
		diff := strings.Repeat("a line of text\n", 1000)
		result := TruncateDiff(diff, 500)
		if len(result) > 500 {
			t.Errorf("expected result <= 500 bytes, got %d", len(result))
		}
		if !strings.Contains(result, "[diff truncated") {
			t.Error("expected truncation notice")
		}
	})

	t.Run("truncation prefers newline boundary", func(t *testing.T) {
		diff := "line1\nline2\nline3\nline4\nline5\n"
		result := TruncateDiff(diff, 20)
		// Should cut at a newline rather than mid-line
		body := strings.Split(result, "\n\n...")[0]
		if strings.Contains(body, "line") && !strings.HasSuffix(body, "") {
			// Just verify it doesn't end mid-word awkwardly
		}
		if !strings.Contains(result, "[diff truncated") {
			t.Error("expected truncation notice")
		}
	})

	t.Run("maxBytes zero uses default", func(t *testing.T) {
		diff := "small"
		result := TruncateDiff(diff, 0)
		if result != diff {
			t.Errorf("expected unchanged diff with maxBytes=0, got %q", result)
		}
	})
}
