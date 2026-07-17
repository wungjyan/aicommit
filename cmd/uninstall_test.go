package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestUninstallRemovesBinaryAndKeepsConfigByDefault(t *testing.T) {
	deps, out, _ := testDeps()
	uninstaller := &fakeUninstaller{result: UninstallResult{Executable: "/Users/test/.local/bin/aicommit"}}
	deps.Uninstaller = uninstaller

	if err := execute(NewRootCommand(deps), "uninstall"); err != nil {
		t.Fatalf("uninstall returned error: %v", err)
	}
	if got, want := len(uninstaller.purges), 1; got != want || uninstaller.purges[0] {
		t.Errorf("purges = %v, want [false]", uninstaller.purges)
	}
	if got := out.String(); !strings.Contains(got, "Removed aicommit binary") || !strings.Contains(got, "Configuration was kept") {
		t.Errorf("unexpected output:\n%s", got)
	}
}

func TestUninstallPurgeConfirmsBeforeRemovingConfig(t *testing.T) {
	deps, out, _ := testDeps()
	deps.In = strings.NewReader("yes\n")
	uninstaller := &fakeUninstaller{result: UninstallResult{
		Executable:    "/Users/test/.local/bin/aicommit",
		ConfigDir:     "/Users/test/.aicommit",
		ConfigRemoved: true,
	}}
	deps.Uninstaller = uninstaller

	if err := execute(NewRootCommand(deps), "uninstall", "--purge"); err != nil {
		t.Fatalf("uninstall --purge returned error: %v", err)
	}
	if got, want := len(uninstaller.purges), 1; got != want || !uninstaller.purges[0] {
		t.Errorf("purges = %v, want [true]", uninstaller.purges)
	}
	if got := out.String(); !strings.Contains(got, "Also remove ~/.aicommit") || !strings.Contains(got, "Removed configuration") {
		t.Errorf("unexpected output:\n%s", got)
	}
}

func TestUninstallPurgeCanBeCancelled(t *testing.T) {
	deps, out, _ := testDeps()
	deps.In = strings.NewReader("no\n")
	uninstaller := &fakeUninstaller{}
	deps.Uninstaller = uninstaller

	if err := execute(NewRootCommand(deps), "uninstall", "--purge"); err != nil {
		t.Fatalf("uninstall --purge returned error: %v", err)
	}
	if len(uninstaller.purges) != 0 {
		t.Errorf("Uninstall called with %v, want no calls", uninstaller.purges)
	}
	if got := out.String(); !strings.Contains(got, "Uninstall cancelled.") {
		t.Errorf("unexpected output:\n%s", got)
	}
}

func TestUninstallPurgeNonInteractiveRequiresYes(t *testing.T) {
	deps, _, _ := testDeps()
	deps.IsTTY = func(any) bool { return false }

	err := execute(NewRootCommand(deps), "uninstall", "--purge")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v, want usage error", err)
	}
}

func TestUninstallPurgeYesSkipsConfirmation(t *testing.T) {
	deps, out, _ := testDeps()
	deps.IsTTY = func(any) bool { return false }
	uninstaller := &fakeUninstaller{result: UninstallResult{Executable: "/Users/test/.local/bin/aicommit"}}
	deps.Uninstaller = uninstaller

	if err := execute(NewRootCommand(deps), "uninstall", "--purge", "--yes"); err != nil {
		t.Fatalf("uninstall --purge --yes returned error: %v", err)
	}
	if got, want := len(uninstaller.purges), 1; got != want || !uninstaller.purges[0] {
		t.Errorf("purges = %v, want [true]", uninstaller.purges)
	}
	if strings.Contains(out.String(), "Also remove ~/.aicommit") {
		t.Errorf("unexpected confirmation prompt:\n%s", out.String())
	}
}

func TestUninstallPropagatesRemovalFailure(t *testing.T) {
	deps, _, _ := testDeps()
	deps.Uninstaller = &fakeUninstaller{err: errBoom}

	if err := execute(NewRootCommand(deps), "uninstall"); !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want %v", err, errBoom)
	}
}
