package ai

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/wungjyan/aicommit/internal/config"
)

var (
	// ErrUnknownBackend indicates a configured backend this build does not know.
	// The factory never falls back to OpenAI on an unknown backend.
	ErrUnknownBackend = errors.New("unknown AI backend")

	// ErrBackendUnavailable indicates a known backend whose implementation is
	// not wired up in this build yet.
	ErrBackendUnavailable = errors.New("AI backend not available")
)

// NewProvider builds the Provider selected by the effective configuration.
//
// It resolves the backend through config.Resolve so the factory and the config
// display command share one source of truth. Unknown backends return
// ErrUnknownBackend; known-but-unimplemented backends return
// ErrBackendUnavailable. Neither ever falls back to OpenAI.
func NewProvider(cfg config.Config) (Provider, error) {
	eff := config.Resolve(cfg)

	switch eff.Backend.Value {
	case config.BackendOpenAI:
		return newOpenAIFromEffective(eff)
	case config.BackendCodex, config.BackendClaude:
		return nil, fmt.Errorf("%w: %q is planned but not implemented in this build", ErrBackendUnavailable, eff.Backend.Value)
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
