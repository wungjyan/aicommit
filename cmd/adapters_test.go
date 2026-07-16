package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestProductionDepsWritesUIToStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := productionDeps(VersionInfo{}, strings.NewReader(""), &stdout, &stderr)

	deps.UI.Info("diagnostic")

	if stdout.Len() != 0 {
		t.Errorf("stdout received diagnostic output: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "diagnostic") {
		t.Errorf("stderr missing diagnostic output: %q", stderr.String())
	}
}
