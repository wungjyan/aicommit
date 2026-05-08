package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wungjyan/aicommit/internal/config"
)

var (
	ErrAPIKeyInvalid = errors.New("invalid API key — check your OPENAI_API_KEY")
	ErrRateLimited   = errors.New("rate limited by OpenAI — try again later")
	ErrRequestFailed = errors.New("request to AI provider failed")
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o-mini"
	defaultTimeout = 60 * time.Second
)

// OpenAIProvider implements Provider using OpenAI's chat completions API.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider.
// Priority: environment variables > cfg > built-in defaults.
// Returns ErrNotConfigured when no API key is available from either source.
var ErrNotConfigured = errors.New("API key not configured — run `aicommit ai` to set up")

func NewOpenAIProvider(cfg config.Config) (*OpenAIProvider, error) {
	apiKey := envOr("OPENAI_API_KEY", cfg.APIKey)
	if apiKey == "" {
		return nil, ErrNotConfigured
	}

	baseURL := envOr("OPENAI_BASE_URL", cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	model := envOr("OPENAI_MODEL", cfg.Model)
	if model == "" {
		model = defaultModel
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}, nil
}

func envOr(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

// isTimeoutError checks whether an error (possibly wrapped in url.Error) is a timeout.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	type timeoutErr interface {
		Timeout() bool
	}
	var te timeoutErr
	if errors.As(err, &te) {
		return te.Timeout()
	}
	return false
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Ping verifies connectivity and API key validity by sending a minimal
// /chat/completions request (the same endpoint used by Generate).
// This is more compatible than GET /models, which many OpenAI-compatible
// services do not implement.
func (p *OpenAIProvider) Ping(ctx context.Context) error {
	reqBody := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "user", Content: "ping"},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach %s: %w", p.baseURL, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrAPIKeyInvalid
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		respBody, _ := io.ReadAll(resp.Body)
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, p.baseURL, preview)
	}
}

func (p *OpenAIProvider) Generate(ctx context.Context, diff string) (string, error) {
	reqBody := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: diff,
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || isTimeoutError(err) {
			return "", fmt.Errorf("AI request timed out (waited %s) — the model may be slow, please retry: %w", defaultTimeout, ErrRequestFailed)
		}
		return "", fmt.Errorf("failed to call AI API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return "", ErrAPIKeyInvalid
		case http.StatusTooManyRequests:
			return "", ErrRateLimited
		default:
			return "", fmt.Errorf("AI API error (status %d): %s", resp.StatusCode, string(respBody))
		}
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return "", fmt.Errorf("failed to parse response (got: %s): %w", preview, err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no commit message generated")
	}

	message := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// Strip surrounding quotes if present
	message = strings.Trim(message, "`\"'")

	return message, nil
}

var systemPrompt = `You are a commit message generator. Given a git diff, generate a Conventional Commit message.

## Format

<type>[optional scope]: <description>

[optional body]

## Rules

- Follow the Conventional Commits specification strictly.
- The first line (header) MUST be ≤ 72 characters.
- Use imperative mood in the description: "add", "fix", "update" (not "added", "fixed", "updates").
- Do not capitalise the first letter of the description.
- Do not end the description with a period.
- Scope is optional. Use it when it adds clarity (e.g. the name of a module, component, or file).
- Choose the type that best represents the dominant change:
  - feat     — a new feature
  - fix      — a bug fix
  - docs     — documentation only
  - style    — formatting, whitespace (no logic change)
  - refactor — code restructuring (no feature or fix)
  - perf     — performance improvement
  - test     — adding or updating tests
  - chore    — build process, dependencies, tooling
  - ci       — CI/CD configuration
  - build    — build system or external dependencies
- When the diff touches multiple areas, focus on the most impactful change.
- Use a footer only for breaking changes: BREAKING CHANGE: <description>
- Output ONLY the commit message. No markdown fences, no explanation, no extra text.

## Examples

feat(auth): add JWT refresh token validation
fix(api): handle null response from user endpoint
docs(readme): update installation instructions
refactor(utils): extract common validation logic
test(auth): add unit tests for login flow
fix: resolve race condition in worker pool shutdown

chore(deps): bump axios from 1.6.0 to 1.7.2

BREAKING CHANGE: the --legacy flag has been removed`
