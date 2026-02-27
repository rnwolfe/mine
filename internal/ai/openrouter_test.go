package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenRouterProvider_Name(t *testing.T) {
	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       &http.Client{},
	}
	if p.Name() != "openrouter" {
		t.Errorf("expected 'openrouter', got '%s'", p.Name())
	}
}

func TestOpenRouterProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header")
		}
		if r.Header.Get("HTTP-Referer") == "" {
			t.Error("expected HTTP-Referer header")
		}
		if r.Header.Get("X-Title") == "" {
			t.Error("expected X-Title header")
		}

		resp := openRouterResponse{
			ID:    "chatcmpl-abc",
			Model: "openai/gpt-4",
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{Role: "assistant", Content: "Hello from OpenRouter!"}},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     6,
				CompletionTokens: 8,
				TotalTokens:      14,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Hello from OpenRouter!" {
		t.Errorf("expected 'Hello from OpenRouter!', got '%s'", resp.Content)
	}
	if resp.Usage.TotalTokens != 14 {
		t.Errorf("expected 14 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestOpenRouterProvider_Complete_WithSystem(t *testing.T) {
	var receivedBody openRouterRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		resp := openRouterResponse{
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{Content: "ok"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.System = "You are an assistant"
	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// System message should be prepended as first message
	if len(receivedBody.Messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(receivedBody.Messages))
	}
	if receivedBody.Messages[0].Role != "system" {
		t.Errorf("expected first message role 'system', got '%s'", receivedBody.Messages[0].Role)
	}
	if receivedBody.Messages[0].Content != "You are an assistant" {
		t.Errorf("expected system content, got '%s'", receivedBody.Messages[0].Content)
	}
}

func TestOpenRouterProvider_Complete_WithModelOverride(t *testing.T) {
	var receivedBody openRouterRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		resp := openRouterResponse{Model: "anthropic/claude-3"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.Model = "anthropic/claude-3"
	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if receivedBody.Model != "anthropic/claude-3" {
		t.Errorf("expected model 'anthropic/claude-3', got '%s'", receivedBody.Model)
	}
}

func TestOpenRouterProvider_Complete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openRouterResponse{Model: "openai/gpt-4", Choices: nil}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("expected empty content for empty choices, got '%s'", resp.Content)
	}
}

func TestOpenRouterProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limit exceeded"}`)) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for API error status")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected error to contain status code, got: %v", err)
	}
}

func TestOpenRouterProvider_Complete_ValidationError(t *testing.T) {
	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       &http.Client{},
	}
	req := &Request{
		Prompt:      "Hello",
		MaxTokens:   0, // invalid
		Temperature: 0.5,
	}
	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestOpenRouterProvider_Stream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		events := []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":", Router!"}}]}`,
			`data: [DONE]`,
		}
		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
		}
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if buf.String() != "Hello, Router!" {
		t.Errorf("expected 'Hello, Router!', got '%s'", buf.String())
	}
}

func TestOpenRouterProvider_Stream_WithSystem(t *testing.T) {
	var receivedBody openRouterRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.System = "Be terse"
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if !receivedBody.Stream {
		t.Error("expected stream=true in request body")
	}
	if len(receivedBody.Messages) < 2 || receivedBody.Messages[0].Role != "system" {
		t.Error("expected system message prepended in stream request")
	}
}

func TestOpenRouterProvider_Stream_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`)) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "bad-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err == nil {
		t.Fatal("expected error for API error status")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain status code, got: %v", err)
	}
}

func TestOpenRouterProvider_Stream_ValidationError(t *testing.T) {
	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       &http.Client{},
	}
	req := &Request{
		Prompt:      "Hello",
		MaxTokens:   -1, // invalid
		Temperature: 0.7,
	}
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestOpenRouterProvider_Stream_DoneSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"before"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"after"}}]}`)
	}))
	defer server.Close()

	p := &OpenRouterProvider{
		apiKey:       "test-key",
		defaultModel: "openai/gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if strings.Contains(buf.String(), "after") {
		t.Error("stream should stop at [DONE] signal")
	}
	if !strings.Contains(buf.String(), "before") {
		t.Error("stream should include content before [DONE]")
	}
}

func TestOpenRouterProvider_Init_EmptyAPIKey(t *testing.T) {
	t.Cleanup(resetRegistry)

	Register("openrouter-init-test", func(apiKey string) (Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("API key required for OpenRouter")
		}
		return &OpenRouterProvider{
			apiKey:       apiKey,
			defaultModel: "openai/gpt-4",
			client:       &http.Client{},
		}, nil
	})

	_, err := GetProvider("openrouter-init-test", "")
	if err == nil {
		t.Error("expected error for empty API key")
	}
}
