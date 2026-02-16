package ai

import (
	"context"
	"io"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// CompletionRequest represents a request to an AI provider.
type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Stream      bool
}

// CompletionResponse represents a response from an AI provider.
type CompletionResponse struct {
	Content string
	Usage   Usage
}

// Usage tracks token usage for a request.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// StreamCallback is called for each chunk when streaming.
type StreamCallback func(chunk string) error

// Provider defines the interface that all AI providers must implement.
type Provider interface {
	// Complete sends a request and returns the full response.
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

	// Stream sends a request and calls the callback for each chunk.
	Stream(ctx context.Context, req CompletionRequest, callback StreamCallback) (*CompletionResponse, error)

	// Name returns the provider's name.
	Name() string
}

// ProviderConfig holds common provider configuration.
type ProviderConfig struct {
	APIKey  string
	Model   string
	BaseURL string
	Writer  io.Writer // For streaming output
}
