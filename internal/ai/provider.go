package ai

import (
	"context"
	"io"
)

// Provider defines the interface for AI providers (Claude, OpenAI, Ollama, etc.).
type Provider interface {
	// Name returns the provider's identifier (e.g., "claude", "openai", "ollama").
	Name() string

	// Complete sends a prompt and returns the full response.
	Complete(ctx context.Context, req *Request) (*Response, error)

	// Stream sends a prompt and streams the response to the writer.
	Stream(ctx context.Context, req *Request, w io.Writer) error
}

// Request represents a prompt to send to the AI provider.
type Request struct {
	// Prompt is the main text to send to the AI.
	Prompt string

	// System is an optional system prompt to guide the AI's behavior.
	System string

	// MaxTokens is the maximum number of tokens to generate (0 = provider default).
	MaxTokens int

	// Temperature controls randomness (0.0-1.0, 0 = provider default).
	Temperature float64
}

// Response represents the AI provider's response.
type Response struct {
	// Content is the generated text.
	Content string

	// Model is the model that was used.
	Model string

	// Usage contains token usage statistics if available.
	Usage *Usage
}

// Usage contains token usage statistics.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// ErrNoAPIKey is returned when the provider requires an API key but none is configured.
var ErrNoAPIKey = &ProviderError{
	Provider: "unknown",
	Message:  "API key not configured",
}

// ProviderError wraps provider-specific errors.
type ProviderError struct {
	Provider string
	Message  string
	Err      error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Provider + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Provider + ": " + e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
