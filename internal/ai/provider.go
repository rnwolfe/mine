package ai

import (
	"context"
	"fmt"
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
	// Valid range: [0.0, 1.0]. Values outside this range may cause API errors.
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

// Validate checks if the request has valid parameters.
// Returns an error if any parameter is out of acceptable range.
func (r *Request) Validate() error {
	if r.Temperature < 0.0 || r.Temperature > 1.0 {
		return fmt.Errorf("temperature must be in range [0.0, 1.0], got %f", r.Temperature)
	}
	if r.MaxTokens < 1 {
		return fmt.Errorf("max_tokens must be positive, got %d", r.MaxTokens)
	}
	return nil
}
