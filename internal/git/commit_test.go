package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCommit(t *testing.T) {
	t.Run("successful commit", func(t *testing.T) {
		dir := setupTempGitRepo(t)
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)

		// Create and stage a file
		file := filepath.Join(dir, "test.txt")
		os.WriteFile(file, []byte("content\n"), 0644)
		exec.Command("git", "add", "test.txt").Run()

		err := Commit("feat: initial commit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify commit exists
		cmd := exec.Command("git", "log", "--oneline", "-1")
		cmd.Dir = dir
		out, _ := cmd.Output()
		if !contains(string(out), "feat: initial commit") {
			t.Errorf("expected commit message in log, got %q", string(out))
		}
	})

	t.Run("commit without staged changes fails", func(t *testing.T) {
		dir := setupTempGitRepo(t)
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		os.Chdir(dir)

		err := Commit("feat: empty commit")
		if err == nil {
			t.Error("expected error for commit with no staged changes")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
