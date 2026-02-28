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

func TestClaudeProvider_Name(t *testing.T) {
	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
		client:       &http.Client{},
	}
	if p.Name() != "claude" {
		t.Errorf("expected 'claude', got '%s'", p.Name())
	}
}

func TestClaudeProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			t.Error("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("expected anthropic-version header")
		}

		resp := claudeResponse{
			ID:    "msg_123",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-5",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "Hello, world!"},
			},
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got '%s'", resp.Content)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("expected 20 completion tokens, got %d", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected 30 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestClaudeProvider_Complete_WithSystem(t *testing.T) {
	var receivedBody claudeRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		resp := claudeResponse{
			Model: "claude-sonnet-4-5",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "Response"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.System = "You are a helpful assistant"
	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if receivedBody.System != "You are a helpful assistant" {
		t.Errorf("expected system message in request, got '%s'", receivedBody.System)
	}
}

func TestClaudeProvider_Complete_WithModelOverride(t *testing.T) {
	var receivedBody claudeRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		resp := claudeResponse{Model: "claude-haiku-4-5"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.Model = "claude-haiku-4-5"
	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if receivedBody.Model != "claude-haiku-4-5" {
		t.Errorf("expected model 'claude-haiku-4-5', got '%s'", receivedBody.Model)
	}
}

func TestClaudeProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`)) //nolint:errcheck
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "bad-key",
		defaultModel: "claude-sonnet-4-5",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for API error status")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain status code, got: %v", err)
	}
}

func TestClaudeProvider_Complete_ValidationError(t *testing.T) {
	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
		client:       &http.Client{},
	}
	req := &Request{
		Prompt:      "Hello",
		MaxTokens:   100,
		Temperature: 2.0, // invalid
	}
	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestClaudeProvider_Complete_MultipleContentBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			Model: "claude-sonnet-4-5",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "Hello"},
				{Type: "text", Text: ", world!"},
				{Type: "tool_use", Text: ""}, // non-text block should be skipped
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Hello, world!" {
		t.Errorf("expected concatenated content, got '%s'", resp.Content)
	}
}

func TestClaudeProvider_Stream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		events := []string{
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":", world!"}}`,
			`data: {"type":"message_stop"}`,
		}
		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
		}
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
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

func TestClaudeProvider_Stream_WithSystem(t *testing.T) {
	var receivedBody claudeRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"message_stop"}`)
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.System = "Be helpful"
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if !receivedBody.Stream {
		t.Error("expected stream=true in request body")
	}
	if receivedBody.System != "Be helpful" {
		t.Errorf("expected system message 'Be helpful', got '%s'", receivedBody.System)
	}
}

func TestClaudeProvider_Stream_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`)) //nolint:errcheck
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "bad-key",
		defaultModel: "claude-sonnet-4-5",
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

func TestClaudeProvider_Stream_ValidationError(t *testing.T) {
	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
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

func TestClaudeProvider_Stream_DoneSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"before"}}`)
		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"after"}}`)
	}))
	defer server.Close()

	p := &ClaudeProvider{
		apiKey:       "test-key",
		defaultModel: "claude-sonnet-4-5",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if strings.Contains(buf.String(), "after") {
		t.Error("stream should stop at [DONE] signal, but continued past it")
	}
	if !strings.Contains(buf.String(), "before") {
		t.Error("stream should include content before [DONE]")
	}
}

func TestClaudeProvider_Init_EmptyAPIKey(t *testing.T) {
	// Call the real constructor registered by claude.go init() to verify it
	// rejects an empty key. This ensures the actual production path is covered,
	// not just a synthetic test factory.
	_, err := GetProvider("claude", "")
	if err == nil {
		t.Error("expected error for empty API key")
	}
}
