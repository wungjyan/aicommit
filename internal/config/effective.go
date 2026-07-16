package config

import (
	"os"
	"strings"
)

// Backend identifiers.
const (
	BackendOpenAI = "openai"
	BackendCodex  = "codex"
	BackendClaude = "claude"
)

// Built-in defaults for the OpenAI backend and shared language.
const (
	DefaultBackend  = BackendOpenAI
	DefaultBaseURL  = "https://api.openai.com/v1"
	DefaultModel    = "gpt-4o-mini"
	DefaultLanguage = "English"
)

// Environment variables consulted during resolution.
const (
	EnvBackend  = "AICOMMIT_BACKEND"
	EnvAPIKey   = "OPENAI_API_KEY"
	EnvBaseURL  = "OPENAI_BASE_URL"
	EnvModel    = "OPENAI_MODEL"
	EnvLanguage = "AICOMMIT_LANGUAGE"
)

// Source records where a resolved value came from.
type Source string

const (
	SourceEnv     Source = "environment"
	SourceConfig  Source = "config"
	SourceCLI     Source = "cli"
	SourceDefault Source = "default"
)

// Value is a single resolved setting paired with its origin.
type Value struct {
	Value  string
	Source Source
}

// Effective is the fully resolved configuration after applying the
// environment > config file > default priority. Both the provider factory and
// the `config` display command must consume this same result so their view of
// the configuration never drifts.
//
// Backend and Language apply to every backend. APIKey, BaseURL and Model are
// only meaningful for the OpenAI backend; for Codex and Claude they are left
// zero (Source == "") so callers can tell they do not apply, and the CLI's own
// authentication and model configuration are used instead.
type Effective struct {
	Backend  Value
	Language Value

	APIKey  Value
	BaseURL Value
	Model   Value
}

// IsOpenAI reports whether the effective backend is the OpenAI backend.
func (e Effective) IsOpenAI() bool {
	return e.Backend.Value == BackendOpenAI
}

// KnownBackend reports whether name is a backend this build can operate.
func KnownBackend(name string) bool {
	switch name {
	case BackendOpenAI, BackendCodex, BackendClaude:
		return true
	default:
		return false
	}
}

// Resolve applies the effective-settings priority using the process environment.
func Resolve(cfg Config) Effective {
	return resolveWith(cfg, os.Getenv)
}

// ResolveOpenAI resolves the OpenAI-backend fields (API key, base URL, model)
// and language regardless of the configured backend. It backs the explicit
// OpenAI constructor and connection verification, which always operate on the
// OpenAI backend.
func ResolveOpenAI(cfg Config) Effective {
	return resolveOpenAIWith(cfg, os.Getenv)
}

// resolveWith is the testable core of Resolve; getenv is injected so tests can
// exercise environment precedence without mutating process state.
func resolveWith(cfg Config, getenv func(string) string) Effective {
	backend := resolveField(getenv(EnvBackend), cfg.Backend, DefaultBackend)
	// Normalize the backend identifier but keep its source.
	backend.Value = strings.ToLower(strings.TrimSpace(backend.Value))
	if backend.Value == "" {
		backend = Value{Value: DefaultBackend, Source: SourceDefault}
	}

	language := resolveField(getenv(EnvLanguage), cfg.Language, DefaultLanguage)

	eff := Effective{Backend: backend, Language: language}

	switch backend.Value {
	case BackendOpenAI:
		applyOpenAIFields(&eff, cfg, getenv)
	case BackendCodex, BackendClaude:
		// The model is managed by the local CLI, not by aicommit config.
		eff.Model = Value{Source: SourceCLI}
	}

	return eff
}

// resolveOpenAIWith resolves the OpenAI fields and language, forcing the OpenAI
// backend. Used when the caller already knows it wants the OpenAI backend.
func resolveOpenAIWith(cfg Config, getenv func(string) string) Effective {
	eff := Effective{
		Backend:  Value{Value: BackendOpenAI, Source: SourceDefault},
		Language: resolveField(getenv(EnvLanguage), cfg.Language, DefaultLanguage),
	}
	applyOpenAIFields(&eff, cfg, getenv)
	return eff
}

// applyOpenAIFields fills the API key, base URL and model on eff using the
// env > config > default priority. API-key has no default so a missing key
// stays empty (treated as "not configured").
func applyOpenAIFields(eff *Effective, cfg Config, getenv func(string) string) {
	eff.APIKey = resolveOptional(getenv(EnvAPIKey), cfg.APIKey)

	baseURL := resolveField(getenv(EnvBaseURL), cfg.BaseURL, DefaultBaseURL)
	baseURL.Value = strings.TrimRight(baseURL.Value, "/")
	eff.BaseURL = baseURL

	eff.Model = resolveField(getenv(EnvModel), cfg.Model, DefaultModel)
}

// resolveField picks env > config > default and reports the winning source.
func resolveField(env, cfg, def string) Value {
	if env != "" {
		return Value{Value: env, Source: SourceEnv}
	}
	if cfg != "" {
		return Value{Value: cfg, Source: SourceConfig}
	}
	return Value{Value: def, Source: SourceDefault}
}

// resolveOptional picks env > config with no default. A missing value keeps the
// default source with an empty string, letting callers treat it as "not set"
// (e.g. a missing OpenAI API key) rather than a real value.
func resolveOptional(env, cfg string) Value {
	if env != "" {
		return Value{Value: env, Source: SourceEnv}
	}
	if cfg != "" {
		return Value{Value: cfg, Source: SourceConfig}
	}
	return Value{Source: SourceDefault}
}
