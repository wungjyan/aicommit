package config

import "testing"

// envMap builds a getenv function backed by a map, so tests exercise
// environment precedence without touching real process state.
func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolveDefaultsWhenEmpty(t *testing.T) {
	eff := resolveWith(Config{}, envMap(nil))

	if eff.Backend.Value != BackendOpenAI || eff.Backend.Source != SourceDefault {
		t.Errorf("backend = %+v, want openai/default", eff.Backend)
	}
	if eff.BaseURL.Value != DefaultBaseURL || eff.BaseURL.Source != SourceDefault {
		t.Errorf("baseURL = %+v, want default", eff.BaseURL)
	}
	if eff.Model.Value != DefaultModel || eff.Model.Source != SourceDefault {
		t.Errorf("model = %+v, want default", eff.Model)
	}
	if eff.Language.Value != DefaultLanguage || eff.Language.Source != SourceDefault {
		t.Errorf("language = %+v, want default", eff.Language)
	}
	// Missing API key stays empty so callers can treat it as "not configured".
	if eff.APIKey.Value != "" {
		t.Errorf("apiKey = %q, want empty", eff.APIKey.Value)
	}
}

func TestResolveBackendPriority(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		env        map[string]string
		wantValue  string
		wantSource Source
	}{
		{
			name:       "env overrides config",
			cfg:        Config{Backend: "openai"},
			env:        map[string]string{EnvBackend: "codex"},
			wantValue:  BackendCodex,
			wantSource: SourceEnv,
		},
		{
			name:       "config used when no env",
			cfg:        Config{Backend: "claude"},
			wantValue:  BackendClaude,
			wantSource: SourceConfig,
		},
		{
			name:       "default when neither set",
			cfg:        Config{},
			wantValue:  BackendOpenAI,
			wantSource: SourceDefault,
		},
		{
			name:       "value is normalized to lowercase",
			cfg:        Config{Backend: "  Codex "},
			wantValue:  BackendCodex,
			wantSource: SourceConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eff := resolveWith(tt.cfg, envMap(tt.env))
			if eff.Backend.Value != tt.wantValue || eff.Backend.Source != tt.wantSource {
				t.Errorf("backend = %+v, want %s/%s", eff.Backend, tt.wantValue, tt.wantSource)
			}
		})
	}
}

// A config written before multi-backend support has no backend field; it must
// load as the OpenAI backend so existing API configs keep working.
func TestResolveLegacyConfigMigratesToOpenAI(t *testing.T) {
	legacy := Config{APIKey: "sk-legacy", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat"}
	eff := resolveWith(legacy, envMap(nil))

	if !eff.IsOpenAI() {
		t.Fatalf("legacy config resolved to backend %q, want openai", eff.Backend.Value)
	}
	if eff.APIKey.Value != "sk-legacy" || eff.APIKey.Source != SourceConfig {
		t.Errorf("apiKey = %+v, want sk-legacy/config", eff.APIKey)
	}
	if eff.BaseURL.Value != "https://api.deepseek.com/v1" {
		t.Errorf("baseURL = %q, want deepseek", eff.BaseURL.Value)
	}
}

func TestResolveOpenAIEnvPriority(t *testing.T) {
	cfg := Config{APIKey: "sk-config", BaseURL: "https://cfg/v1", Model: "cfg-model", Language: "中文"}
	env := map[string]string{
		EnvAPIKey:   "sk-env",
		EnvBaseURL:  "https://env/v1/",
		EnvModel:    "env-model",
		EnvLanguage: "English",
	}
	eff := resolveWith(cfg, envMap(env))

	if eff.APIKey.Value != "sk-env" || eff.APIKey.Source != SourceEnv {
		t.Errorf("apiKey = %+v, want sk-env/environment", eff.APIKey)
	}
	// Trailing slash trimmed from the base URL.
	if eff.BaseURL.Value != "https://env/v1" || eff.BaseURL.Source != SourceEnv {
		t.Errorf("baseURL = %+v, want https://env/v1/environment", eff.BaseURL)
	}
	if eff.Model.Value != "env-model" || eff.Model.Source != SourceEnv {
		t.Errorf("model = %+v, want env-model/environment", eff.Model)
	}
	if eff.Language.Value != "English" || eff.Language.Source != SourceEnv {
		t.Errorf("language = %+v, want English/environment", eff.Language)
	}
}

// An empty environment variable must not override a configured value.
func TestResolveEmptyEnvDoesNotOverride(t *testing.T) {
	cfg := Config{APIKey: "sk-config", BaseURL: "https://cfg/v1", Model: "cfg-model", Language: "中文"}
	env := map[string]string{
		EnvAPIKey:   "",
		EnvBaseURL:  "",
		EnvModel:    "",
		EnvLanguage: "",
		EnvBackend:  "",
	}
	eff := resolveWith(cfg, envMap(env))

	if eff.APIKey.Value != "sk-config" || eff.APIKey.Source != SourceConfig {
		t.Errorf("apiKey = %+v, want sk-config/config", eff.APIKey)
	}
	if eff.BaseURL.Value != "https://cfg/v1" || eff.BaseURL.Source != SourceConfig {
		t.Errorf("baseURL = %+v, want config", eff.BaseURL)
	}
	if eff.Model.Value != "cfg-model" || eff.Model.Source != SourceConfig {
		t.Errorf("model = %+v, want config", eff.Model)
	}
	if eff.Language.Value != "中文" || eff.Language.Source != SourceConfig {
		t.Errorf("language = %+v, want 中文/config", eff.Language)
	}
}

// Non-OpenAI backends must not expose API key, base URL or model fields; their
// model is managed by the CLI and API credentials do not apply.
func TestResolveNonOpenAIHidesAPIFields(t *testing.T) {
	for _, backend := range []string{BackendCodex, BackendClaude} {
		t.Run(backend, func(t *testing.T) {
			// Even if stale API fields linger in the config, they must not surface.
			cfg := Config{Backend: backend, APIKey: "sk-stale", BaseURL: "https://stale/v1", Model: "stale", Language: "中文"}
			eff := resolveWith(cfg, envMap(nil))

			if eff.APIKey.Value != "" || eff.APIKey.Source != "" {
				t.Errorf("apiKey = %+v, want zero value", eff.APIKey)
			}
			if eff.BaseURL.Value != "" || eff.BaseURL.Source != "" {
				t.Errorf("baseURL = %+v, want zero value", eff.BaseURL)
			}
			if eff.Model.Value != "" {
				t.Errorf("model value = %q, want empty (managed by CLI)", eff.Model.Value)
			}
			if eff.Model.Source != SourceCLI {
				t.Errorf("model source = %q, want cli", eff.Model.Source)
			}
			// Language still applies to every backend.
			if eff.Language.Value != "中文" || eff.Language.Source != SourceConfig {
				t.Errorf("language = %+v, want 中文/config", eff.Language)
			}
		})
	}
}

func TestKnownBackend(t *testing.T) {
	for _, b := range []string{BackendOpenAI, BackendCodex, BackendClaude} {
		if !KnownBackend(b) {
			t.Errorf("KnownBackend(%q) = false, want true", b)
		}
	}
	for _, b := range []string{"", "gemini", "openai "} {
		if KnownBackend(b) {
			t.Errorf("KnownBackend(%q) = true, want false", b)
		}
	}
}

// A backend set only via environment must still resolve, and stale OpenAI config
// fields must not leak through when that env backend is a CLI backend.
func TestResolveBackendFromEnvWithStaleConfig(t *testing.T) {
	cfg := Config{Backend: BackendOpenAI, APIKey: "sk-openai", BaseURL: "https://o/v1", Model: "gpt"}
	eff := resolveWith(cfg, envMap(map[string]string{EnvBackend: BackendClaude}))

	if eff.Backend.Value != BackendClaude || eff.Backend.Source != SourceEnv {
		t.Errorf("backend = %+v, want claude/environment", eff.Backend)
	}
	if eff.APIKey.Value != "" || eff.BaseURL.Value != "" {
		t.Errorf("expected no OpenAI fields for claude backend, got apiKey=%q baseURL=%q", eff.APIKey.Value, eff.BaseURL.Value)
	}
}
