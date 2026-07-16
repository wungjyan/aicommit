package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wungjyan/aicommit/internal/config"
)

// newTestProvider creates an OpenAIProvider pointing at the given server URL.
func newTestProvider(serverURL string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  "test-key",
		baseURL: serverURL,
		model:   "gpt-4o-mini",
		client:  &http.Client{Timeout: 2 * time.Second},
	}
}

func TestNewOpenAIProvider(t *testing.T) {
	t.Run("no key anywhere returns ErrNotConfigured", func(t *testing.T) {
		os.Unsetenv("OPENAI_API_KEY")
		_, err := NewOpenAIProvider(config.Config{})
		if !errors.Is(err, ErrNotConfigured) {
			t.Errorf("expected ErrNotConfigured, got %v", err)
		}
	})

	t.Run("config key used when env is empty", func(t *testing.T) {
		os.Unsetenv("OPENAI_API_KEY")
		p, err := NewOpenAIProvider(config.Config{APIKey: "cfg-key"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.apiKey != "cfg-key" {
			t.Errorf("expected cfg-key, got %q", p.apiKey)
		}
	})

	t.Run("env overrides config", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "env-key")
		defer os.Unsetenv("OPENAI_API_KEY")
		p, err := NewOpenAIProvider(config.Config{APIKey: "cfg-key"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.apiKey != "env-key" {
			t.Errorf("expected env-key, got %q", p.apiKey)
		}
	})

	t.Run("default model when neither set", func(t *testing.T) {
		os.Unsetenv("OPENAI_MODEL")
		p, err := NewOpenAIProvider(config.Config{APIKey: "k"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.model != config.DefaultModel {
			t.Errorf("expected %q, got %q", config.DefaultModel, p.model)
		}
	})
}

func TestGenerate(t *testing.T) {
	t.Run("successful generation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify auth header
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("expected Bearer token, got %q", r.Header.Get("Authorization"))
			}
			json.NewEncoder(w).Encode(chatResponse{
				Choices: []struct {
					Message chatMessage `json:"message"`
				}{
					{Message: chatMessage{Role: "assistant", Content: "feat: add new feature"}},
				},
			})
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		msg, err := p.Generate(context.Background(), "some diff")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg != "feat: add new feature" {
			t.Errorf("expected 'feat: add new feature', got %q", msg)
		}
	})

	t.Run("strips surrounding quotes", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(chatResponse{
				Choices: []struct {
					Message chatMessage `json:"message"`
				}{
					{Message: chatMessage{Role: "assistant", Content: "\"feat: quoted\""}},
				},
			})
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		msg, err := p.Generate(context.Background(), "diff")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg != "feat: quoted" {
			t.Errorf("expected 'feat: quoted', got %q", msg)
		}
	})

	t.Run("unauthorized returns ErrAPIKeyInvalid", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		_, err := p.Generate(context.Background(), "diff")
		if !errors.Is(err, ErrAPIKeyInvalid) {
			t.Errorf("expected ErrAPIKeyInvalid, got %v", err)
		}
	})

	t.Run("rate limited returns ErrRateLimited", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"Rate limit exceeded"}}`))
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		_, err := p.Generate(context.Background(), "diff")
		if !errors.Is(err, ErrRateLimited) {
			t.Errorf("expected ErrRateLimited, got %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"internal error"}}`))
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		_, err := p.Generate(context.Background(), "diff")
		if err == nil {
			t.Fatal("expected error for 500 status")
		}
		if errors.Is(err, ErrAPIKeyInvalid) || errors.Is(err, ErrRateLimited) {
			t.Errorf("unexpected sentinel error: %v", err)
		}
	})

	t.Run("empty choices", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(chatResponse{Choices: nil})
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		_, err := p.Generate(context.Background(), "diff")
		if err == nil {
			t.Fatal("expected error for empty choices")
		}
	})

	t.Run("API error in response body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(chatResponse{
				Error: &struct {
					Message string `json:"message"`
				}{Message: "something went wrong"},
			})
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		_, err := p.Generate(context.Background(), "diff")
		if err == nil || !strings.Contains(err.Error(), "something went wrong") {
			t.Errorf("expected error containing 'something went wrong', got %v", err)
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := p.Generate(ctx, "diff")
		if err == nil {
			t.Fatal("expected timeout error")
		}
	})
}

func TestPing(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if !strings.Contains(r.URL.Path, "/chat/completions") {
				t.Errorf("expected /chat/completions, got %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(chatResponse{
				Choices: []struct {
					Message chatMessage `json:"message"`
				}{{Message: chatMessage{Role: "assistant", Content: "pong"}}},
			})
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		if err := p.Ping(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unauthorized returns ErrAPIKeyInvalid", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		if err := p.Ping(context.Background()); !errors.Is(err, ErrAPIKeyInvalid) {
			t.Errorf("expected ErrAPIKeyInvalid, got %v", err)
		}
	})

	t.Run("rate limited returns ErrRateLimited", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"Rate limit exceeded"}}`))
		}))
		defer srv.Close()

		p := newTestProvider(srv.URL)
		if err := p.Ping(context.Background()); !errors.Is(err, ErrRateLimited) {
			t.Errorf("expected ErrRateLimited, got %v", err)
		}
	})
}
