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

func TestOpenAIProvider_Name(t *testing.T) {
	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
		client:       &http.Client{},
	}
	if p.Name() != "openai" {
		t.Errorf("expected 'openai', got '%s'", p.Name())
	}
}

func TestOpenAIProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header")
		}

		resp := openAIResponse{
			ID:    "chatcmpl-123",
			Model: "gpt-4",
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{Role: "assistant", Content: "Hello from OpenAI!"}},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     5,
				CompletionTokens: 10,
				TotalTokens:      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Hello from OpenAI!" {
		t.Errorf("expected content 'Hello from OpenAI!', got '%s'", resp.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestOpenAIProvider_Complete_WithSystem(t *testing.T) {
	var receivedBody openAIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		resp := openAIResponse{
			Model: "gpt-4",
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

	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.System = "You are a helpful assistant"
	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// System message should be prepended as first message with role "system"
	if len(receivedBody.Messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(receivedBody.Messages))
	}
	if receivedBody.Messages[0].Role != "system" {
		t.Errorf("expected first message role 'system', got '%s'", receivedBody.Messages[0].Role)
	}
	if receivedBody.Messages[0].Content != "You are a helpful assistant" {
		t.Errorf("expected system content, got '%s'", receivedBody.Messages[0].Content)
	}
	if receivedBody.Messages[1].Role != "user" {
		t.Errorf("expected second message role 'user', got '%s'", receivedBody.Messages[1].Role)
	}
}

func TestOpenAIProvider_Complete_WithModelOverride(t *testing.T) {
	var receivedBody openAIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		resp := openAIResponse{Model: "gpt-3.5-turbo"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.Model = "gpt-3.5-turbo"
	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if receivedBody.Model != "gpt-3.5-turbo" {
		t.Errorf("expected model 'gpt-3.5-turbo', got '%s'", receivedBody.Model)
	}
}

func TestOpenAIProvider_Complete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{Model: "gpt-4", Choices: nil}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
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

func TestOpenAIProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": {"message": "insufficient quota"}}`)) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenAIProvider{
		apiKey:       "bad-key",
		defaultModel: "gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for API error status")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to contain status code, got: %v", err)
	}
}

func TestOpenAIProvider_Complete_ValidationError(t *testing.T) {
	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
		client:       &http.Client{},
	}
	req := &Request{
		Prompt:      "Hello",
		MaxTokens:   0, // invalid
		Temperature: 0.7,
	}
	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestOpenAIProvider_Stream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		events := []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":", world!"}}]}`,
			`data: [DONE]`,
		}
		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
		}
	}))
	defer server.Close()

	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if buf.String() != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got '%s'", buf.String())
	}
}

func TestOpenAIProvider_Stream_WithSystem(t *testing.T) {
	var receivedBody openAIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.System = "Be concise"
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

func TestOpenAIProvider_Stream_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`)) //nolint:errcheck
	}))
	defer server.Close()

	p := &OpenAIProvider{
		apiKey:       "bad-key",
		defaultModel: "gpt-4",
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

func TestOpenAIProvider_Stream_ValidationError(t *testing.T) {
	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
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

func TestOpenAIProvider_Stream_DoneSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"before"}}]}`)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"after"}}]}`)
	}))
	defer server.Close()

	p := &OpenAIProvider{
		apiKey:       "test-key",
		defaultModel: "gpt-4",
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

func TestOpenAIProvider_Init_EmptyAPIKey(t *testing.T) {
	t.Cleanup(resetRegistry)

	Register("openai-init-test", func(apiKey string) (Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("API key required for OpenAI provider")
		}
		return &OpenAIProvider{
			apiKey:       apiKey,
			defaultModel: "gpt-4",
			client:       &http.Client{},
		}, nil
	})

	_, err := GetProvider("openai-init-test", "")
	if err == nil {
		t.Error("expected error for empty API key")
	}
}
