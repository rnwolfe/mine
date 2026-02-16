package ai

import (
	"context"
	"io"
	"strings"
	"testing"
)

// mockProvider is a test implementation of the Provider interface.
type mockProvider struct {
	name        string
	response    string
	streamChunks []string
	err         error
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Complete(_ context.Context, req *Request) (*Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Response{
		Content: m.response,
		Model:   "mock-model",
		Usage: &Usage{
			InputTokens:  len(req.Prompt),
			OutputTokens: len(m.response),
		},
	}, nil
}

func (m *mockProvider) Stream(_ context.Context, _ *Request, w io.Writer) error {
	if m.err != nil {
		return m.err
	}
	for _, chunk := range m.streamChunks {
		if _, err := w.Write([]byte(chunk)); err != nil {
			return err
		}
	}
	return nil
}

func TestMockProvider(t *testing.T) {
	t.Run("complete success", func(t *testing.T) {
		provider := &mockProvider{
			name:     "test",
			response: "Hello, world!",
		}

		req := &Request{
			Prompt: "Say hello",
		}

		resp, err := provider.Complete(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Content != "Hello, world!" {
			t.Errorf("expected 'Hello, world!', got %q", resp.Content)
		}

		if resp.Model != "mock-model" {
			t.Errorf("expected 'mock-model', got %q", resp.Model)
		}
	})

	t.Run("complete error", func(t *testing.T) {
		expectedErr := &ProviderError{Provider: "test", Message: "test error"}
		provider := &mockProvider{
			name: "test",
			err:  expectedErr,
		}

		req := &Request{
			Prompt: "Say hello",
		}

		_, err := provider.Complete(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("stream success", func(t *testing.T) {
		provider := &mockProvider{
			name: "test",
			streamChunks: []string{
				"Hello",
				", ",
				"world!",
			},
		}

		req := &Request{
			Prompt: "Say hello",
		}

		var buf strings.Builder
		err := provider.Stream(context.Background(), req, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		want := "Hello, world!"
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})

	t.Run("stream error", func(t *testing.T) {
		expectedErr := &ProviderError{Provider: "test", Message: "test error"}
		provider := &mockProvider{
			name: "test",
			err:  expectedErr,
		}

		req := &Request{
			Prompt: "Say hello",
		}

		var buf strings.Builder
		err := provider.Stream(context.Background(), req, &buf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestProviderError(t *testing.T) {
	t.Run("error with wrapped error", func(t *testing.T) {
		innerErr := io.EOF
		err := &ProviderError{
			Provider: "test",
			Message:  "read failed",
			Err:      innerErr,
		}

		want := "test: read failed: EOF"
		if got := err.Error(); got != want {
			t.Errorf("expected %q, got %q", want, got)
		}

		if err.Unwrap() != innerErr {
			t.Errorf("expected unwrapped error to be %v, got %v", innerErr, err.Unwrap())
		}
	})

	t.Run("error without wrapped error", func(t *testing.T) {
		err := &ProviderError{
			Provider: "test",
			Message:  "API key missing",
		}

		want := "test: API key missing"
		if got := err.Error(); got != want {
			t.Errorf("expected %q, got %q", want, got)
		}

		if err.Unwrap() != nil {
			t.Errorf("expected unwrapped error to be nil, got %v", err.Unwrap())
		}
	})
}
