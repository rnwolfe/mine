package ai

import (
	"context"
	"io"
)

// Provider is the interface that all AI providers must implement.
type Provider interface {
	// Name returns the provider name (e.g., "claude", "openai")
	Name() string

	// Complete sends a prompt and returns the complete response.
	Complete(ctx context.Context, req *Request) (*Response, error)

	// Stream sends a prompt and streams the response token by token.
	Stream(ctx context.Context, req *Request, w io.Writer) error
}

// Request represents an AI completion request.
type Request struct {
	// Prompt is the user's input text.
	Prompt string

	// System is an optional system message to set context.
	System string

	// Model is an optional model override (if empty, uses provider default).
	Model string

	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int

	// Temperature controls randomness (0.0 = deterministic, 1.0 = creative).
	Temperature float64
}

// Response represents an AI completion response.
type Response struct {
	// Content is the generated text.
	Content string

	// Model is the actual model that was used.
	Model string

	// Usage contains token counts.
	Usage Usage
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// NewRequest creates a request with sensible defaults.
func NewRequest(prompt string) *Request {
	return &Request{
		Prompt:      prompt,
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}
