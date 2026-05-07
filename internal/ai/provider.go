package ai

import "context"

// Provider defines the interface for AI commit message generation.
type Provider interface {
	// Generate takes a git diff and returns a Conventional Commit message.
	Generate(ctx context.Context, diff string) (string, error)
}
