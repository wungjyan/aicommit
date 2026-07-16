package ai

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/wungjyan/aicommit/internal/config"
)

// ErrUnknownBackend indicates a configured backend this build does not know.
// The factory never falls back to OpenAI on an unknown backend.
var ErrUnknownBackend = errors.New("unknown AI backend")

// NewProvider builds the Provider selected by the effective configuration.
//
// It resolves the backend through config.Resolve so the factory and the config
// display command share one source of truth. Unknown backends return
// ErrUnknownBackend and never fall back to OpenAI.
func NewProvider(cfg config.Config) (Provider, error) {
	eff := config.Resolve(cfg)

	switch eff.Backend.Value {
	case config.BackendOpenAI:
		return newOpenAIFromEffective(eff)
	case config.BackendCodex:
		return NewCodexProvider(eff.Language.Value), nil
	case config.BackendClaude:
		return NewClaudeProvider(eff.Language.Value), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownBackend, eff.Backend.Value)
	}
}

// newOpenAIFromEffective assembles an OpenAIProvider from already-resolved
// effective values, so backend dispatch and field resolution stay unified.
func newOpenAIFromEffective(eff config.Effective) (*OpenAIProvider, error) {
	if eff.APIKey.Value == "" {
		return nil, ErrNotConfigured
	}
	return &OpenAIProvider{
		apiKey:   eff.APIKey.Value,
		baseURL:  eff.BaseURL.Value,
		model:    eff.Model.Value,
		language: eff.Language.Value,
		client:   &http.Client{Timeout: defaultTimeout},
	}, nil
}
