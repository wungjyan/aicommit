package ui

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestUIWritesStatusAndProgressToInjectedWriter(t *testing.T) {
	var diagnostics bytes.Buffer
	u := newUI(&diagnostics, func(io.Writer) bool { return false })

	u.Success("saved")
	if err := u.Spinner("Generating", func() error { return nil }); err != nil {
		t.Fatalf("Spinner returned error: %v", err)
	}

	got := diagnostics.String()
	for _, want := range []string{"✔ saved", "Generating... done"} {
		if !strings.Contains(got, want) {
			t.Errorf("diagnostics missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\033[") {
		t.Errorf("non-terminal diagnostics contain ANSI escapes: %q", got)
	}
}
