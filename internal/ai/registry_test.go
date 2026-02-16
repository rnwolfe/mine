package ai

import (
	"context"
	"errors"
	"io"
	"testing"
)

// Mock provider for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Complete(ctx context.Context, req *Request) (*Response, error) {
	return &Response{
		Content: "mock response",
		Model:   "mock-model",
		Usage:   Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}, nil
}

func (m *mockProvider) Stream(ctx context.Context, req *Request, w io.Writer) error {
	_, err := w.Write([]byte("mock stream"))
	return err
}

func TestRegisterAndGetProvider(t *testing.T) {
	// Register a mock provider
	Register("mock", func(apiKey string) (Provider, error) {
		if apiKey == "" {
			return nil, errors.New("API key required")
		}
		return &mockProvider{name: "mock"}, nil
	})

	// Test successful retrieval
	provider, err := GetProvider("mock", "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if provider.Name() != "mock" {
		t.Errorf("expected provider name 'mock', got '%s'", provider.Name())
	}

	// Test missing API key
	_, err = GetProvider("mock", "")
	if err == nil {
		t.Error("expected error for missing API key")
	}

	// Test unknown provider
	_, err = GetProvider("unknown", "test-key")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestListProviders(t *testing.T) {
	// Register providers for this test
	Register("mock", func(apiKey string) (Provider, error) {
		return &mockProvider{name: "mock"}, nil
	})
	Register("mock2", func(apiKey string) (Provider, error) {
		return &mockProvider{name: "mock2"}, nil
	})

	providers := ListProviders()

	// Should have at least 1 provider (including our registered mock2)
	if len(providers) < 1 {
		t.Errorf("expected at least 1 provider, got %d", len(providers))
	}

	// Check for our mock2 provider
	hasMock2 := false
	for _, p := range providers {
		if p == "mock2" {
			hasMock2 = true
		}
	}

	if !hasMock2 {
		t.Error("expected 'mock2' in provider list")
	}
}

func TestProviderComplete(t *testing.T) {
	Register("test-complete", func(apiKey string) (Provider, error) {
		return &mockProvider{name: "test-complete"}, nil
	})

	provider, err := GetProvider("test-complete", "key")
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	req := NewRequest("test prompt")
	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Content != "mock response" {
		t.Errorf("expected content 'mock response', got '%s'", resp.Content)
	}

	if resp.Model != "mock-model" {
		t.Errorf("expected model 'mock-model', got '%s'", resp.Model)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected total tokens 30, got %d", resp.Usage.TotalTokens)
	}
}
