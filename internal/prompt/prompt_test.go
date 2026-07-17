package prompt

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{"valid feat", "feat: add user login", false},
		{"valid with scope", "fix(api): handle null response", false},
		{"valid breaking change", "feat(auth)!: remove legacy token endpoint", false},
		{"valid multi-line", "fix: resolve race condition\n\nThe worker pool could deadlock on shutdown when tasks were still pending.", false},
		{"valid docs", "docs(readme): update installation instructions", false},
		{"valid chore", "chore(deps): bump axios from 1.6.0 to 1.7.2", false},

		{"empty message", "", true},
		{"whitespace only", "   \n  \n ", true},
		{"no type", "add user login", true},
		{"unknown type", "feature: add login", true},
		{"missing description", "fix: ", true},
		{"no space after colon", "fix:resolve bug", true},
		{"header too long", "feat(scope): " + string(make([]byte, 80)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage(%q) error = %v, wantErr %v", tt.message, err, tt.wantErr)
			}
		})
	}
}

func TestEditMessage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	t.Run("editor with arguments", func(t *testing.T) {
		os.Setenv("EDITOR", "sed -i '' s/original/edited/")
		defer os.Unsetenv("EDITOR")

		result, err := EditMessage(strings.NewReader(""), io.Discard, io.Discard, "original text")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "edited text" {
			t.Errorf("expected 'edited text', got %q", result)
		}
	})

	t.Run("simple editor", func(t *testing.T) {
		os.Setenv("EDITOR", "true")
		defer os.Unsetenv("EDITOR")

		result, err := EditMessage(strings.NewReader(""), io.Discard, io.Discard, "unchanged text")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "unchanged text" {
			t.Errorf("expected 'unchanged text', got %q", result)
		}
	})
}

type plainStyle struct{}

func (plainStyle) Bold(s string) string      { return s }
func (plainStyle) Highlight(s string) string { return s }

func TestConfirmUsesInjectedStreams(t *testing.T) {
	var output bytes.Buffer
	action, _, err := Confirm(strings.NewReader("invalid\nq\n"), &output, plainStyle{}, "feat: add output routing", true)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if action != "quit" {
		t.Errorf("action = %q, want quit", action)
	}
	got := output.String()
	if !strings.Contains(got, "Generated commit message:") || !strings.Contains(got, "Invalid input") {
		t.Errorf("injected output missing confirmation diagnostics:\n%s", got)
	}
}

func TestConfirmInvalidMessageEnterOpensEditorInsteadOfRegenerating(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	t.Setenv("EDITOR", "true")
	var output bytes.Buffer
	action, edited, err := Confirm(strings.NewReader("\n"), &output, plainStyle{}, "not a commit message", false)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if action != "edit" {
		t.Errorf("action = %q, want edit", action)
	}
	if edited != "not a commit message" {
		t.Errorf("edited = %q, want original message", edited)
	}
	if !strings.Contains(output.String(), "[Enter/e] edit") {
		t.Errorf("confirmation prompt does not show Enter edit:\n%s", output.String())
	}
}
