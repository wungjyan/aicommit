package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// withTestHome points HOME (and USERPROFILE on Windows) at a temp dir so config
// reads and writes never touch the real user directory.
func withTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	return home
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	withTestHome(t)

	want := Config{Backend: BackendCodex, APIKey: "sk-abc", BaseURL: "https://x/v1", Model: "m", Language: "中文"}
	if err := SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got != want {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestSaveConfigPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits not meaningful on Windows")
	}
	home := withTestHome(t)

	if err := SaveConfig(Config{APIKey: "sk-x"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	dir := filepath.Join(home, configDir)
	di, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := di.Mode().Perm(); perm != 0700 {
		t.Errorf("config dir perm = %o, want 0700", perm)
	}

	fi, err := os.Stat(filepath.Join(dir, configFile))
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Errorf("config file perm = %o, want 0600", perm)
	}
}

// A failed marshal-free write path leaves no stray temp files behind, and a
// successful save does not litter the directory.
func TestSaveConfigNoTempLeftovers(t *testing.T) {
	home := withTestHome(t)
	if err := SaveConfig(Config{APIKey: "sk-x"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(home, configDir))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != configFile {
			t.Errorf("unexpected leftover file: %s", e.Name())
		}
	}
}

// Overwriting an existing config replaces it atomically without corruption.
func TestSaveConfigOverwrite(t *testing.T) {
	withTestHome(t)

	if err := SaveConfig(Config{APIKey: "old"}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := SaveConfig(Config{APIKey: "new", Model: "m2"}); err != nil {
		t.Fatalf("second save: %v", err)
	}
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.APIKey != "new" || got.Model != "m2" {
		t.Errorf("overwrite failed: %+v", got)
	}
}

func TestConfigPathUsesHome(t *testing.T) {
	home := withTestHome(t)
	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}
	want := filepath.Join(home, configDir, configFile)
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}
