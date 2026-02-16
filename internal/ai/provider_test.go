package ai

import (
	"context"
	"testing"
	"time"
)

// MockProvider implements Provider for testing.
type MockProvider struct {
	name     string
	response *CompletionResponse
	err      error
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *MockProvider) Stream(ctx context.Context, req CompletionRequest, callback StreamCallback) (*CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Simulate streaming by calling callback with chunks
	if callback != nil {
		chunks := []string{"Hello", " ", "world"}
		for _, chunk := range chunks {
			if err := callback(chunk); err != nil {
				return nil, err
			}
		}
	}
	return m.response, nil
}

func TestMockProvider(t *testing.T) {
	provider := &MockProvider{
		name: "mock",
		response: &CompletionResponse{
			Content: "Hello world",
			Usage: Usage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
	}

	// Test Complete
	resp, err := provider.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", resp.Content)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.Usage.InputTokens)
	}

	// Test Stream
	var chunks []string
	_, err = provider.Stream(ctx, req, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(chunks))
	}

	// Test Name
	if provider.Name() != "mock" {
		t.Errorf("expected 'mock', got %q", provider.Name())
	}
}

func TestCompletionRequest(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		MaxTokens:   1000,
		Temperature: 0.8,
		Stream:      true,
	}

	if len(req.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Errorf("expected first message role to be 'user', got %q", req.Messages[0].Role)
	}
	if req.MaxTokens != 1000 {
		t.Errorf("expected max tokens 1000, got %d", req.MaxTokens)
	}
	if req.Temperature != 0.8 {
		t.Errorf("expected temperature 0.8, got %f", req.Temperature)
	}
}
