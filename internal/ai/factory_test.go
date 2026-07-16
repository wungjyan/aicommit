package ai

import (
	"errors"
	"testing"

	"github.com/wungjyan/aicommit/internal/config"
)

func TestNewProviderOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("AICOMMIT_BACKEND", "")

	p, err := NewProvider(config.Config{Backend: config.BackendOpenAI, APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*OpenAIProvider); !ok {
		t.Errorf("expected *OpenAIProvider, got %T", p)
	}
}

// A config with no backend field must resolve to OpenAI (migration behavior).
func TestNewProviderLegacyConfigIsOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("AICOMMIT_BACKEND", "")

	p, err := NewProvider(config.Config{APIKey: "sk-legacy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*OpenAIProvider); !ok {
		t.Errorf("expected *OpenAIProvider, got %T", p)
	}
}

func TestNewProviderOpenAIMissingKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("AICOMMIT_BACKEND", "")

	_, err := NewProvider(config.Config{Backend: config.BackendOpenAI})
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("expected ErrNotConfigured, got %v", err)
	}
}

// Codex is implemented: the factory returns a CodexProvider without requiring
// the CLI to be installed at construction time.
func TestNewProviderCodex(t *testing.T) {
	t.Setenv("AICOMMIT_BACKEND", "")

	p, err := NewProvider(config.Config{Backend: config.BackendCodex})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*CodexProvider); !ok {
		t.Errorf("expected *CodexProvider, got %T", p)
	}
}

// Claude is implemented: the factory returns a ClaudeProvider without requiring
// the CLI to be installed at construction time.
func TestNewProviderClaude(t *testing.T) {
	t.Setenv("AICOMMIT_BACKEND", "")

	p, err := NewProvider(config.Config{Backend: config.BackendClaude})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*ClaudeProvider); !ok {
		t.Errorf("expected *ClaudeProvider, got %T", p)
	}
}

// An unknown backend must return ErrUnknownBackend, not silently use OpenAI.
func TestNewProviderUnknownBackend(t *testing.T) {
	t.Setenv("AICOMMIT_BACKEND", "")

	_, err := NewProvider(config.Config{Backend: "gemini", APIKey: "sk-test"})
	if !errors.Is(err, ErrUnknownBackend) {
		t.Errorf("expected ErrUnknownBackend, got %v", err)
	}
}

// AICOMMIT_BACKEND overrides the config file backend in the factory.
func TestNewProviderBackendEnvOverride(t *testing.T) {
	t.Setenv("AICOMMIT_BACKEND", "codex")

	// Config says openai, but the env selects codex.
	p, err := NewProvider(config.Config{Backend: config.BackendOpenAI, APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*CodexProvider); !ok {
		t.Errorf("expected env backend codex -> *CodexProvider, got %T", p)
	}
}
