package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	defaultTimeout = 60 * time.Second
)

// OpenAIProvider implements Provider using OpenAI's chat completions API.
type OpenAIProvider struct {
	apiKey   string
	baseURL  string
	model    string
	language string
	client   *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider on the OpenAI backend.
// Priority: environment variables > cfg > built-in defaults (via config.ResolveOpenAI).
// Returns ErrNotConfigured when no API key is available from either source.
var ErrNotConfigured = errors.New("API key not configured — run `aicommit config setup` to set up")

func NewOpenAIProvider(cfg config.Config) (*OpenAIProvider, error) {
	return newOpenAIFromEffective(config.ResolveOpenAI(cfg))
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
		return fmt.Errorf("cannot reach %s: %w", p.baseURL, requestFailed(err))
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
		return fmt.Errorf("%w: unexpected status %d from %s: %s", ErrRequestFailed, resp.StatusCode, p.baseURL, preview)
	}
}

func (p *OpenAIProvider) Generate(ctx context.Context, diff string) (string, error) {
	prompt := BuildPrompt(p.language)

	reqBody := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: prompt,
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
		return "", fmt.Errorf("failed to call AI API: %w", requestFailed(err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", requestFailed(err))
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return "", ErrAPIKeyInvalid
		case http.StatusTooManyRequests:
			return "", ErrRateLimited
		default:
			return "", fmt.Errorf("%w: AI API error (status %d): %s", ErrRequestFailed, resp.StatusCode, string(respBody))
		}
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return "", fmt.Errorf("failed to parse response (got: %s): %w", preview, requestFailed(err))
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("%w: OpenAI API error: %s", ErrRequestFailed, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("%w: no commit message generated", ErrRequestFailed)
	}

	message := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// Strip surrounding quotes if present
	message = strings.Trim(message, "`\"'")

	return message, nil
}

func requestFailed(err error) error {
	return fmt.Errorf("%w: %w", ErrRequestFailed, err)
}
