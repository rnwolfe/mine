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

func TestGeminiProvider_Name(t *testing.T) {
	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       &http.Client{},
	}
	if p.Name() != "gemini" {
		t.Errorf("expected 'gemini', got '%s'", p.Name())
	}
}

func TestGeminiProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []geminiPart `json:"parts"`
				} `json:"content"`
			}{
				{Content: struct {
					Parts []geminiPart `json:"parts"`
				}{Parts: []geminiPart{{Text: "Hello from Gemini!"}}}},
			},
			UsageMetadata: &struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			}{
				PromptTokenCount:     8,
				CandidatesTokenCount: 12,
				TotalTokenCount:      20,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Hello from Gemini!" {
		t.Errorf("expected 'Hello from Gemini!', got '%s'", resp.Content)
	}
	if resp.Usage.PromptTokens != 8 {
		t.Errorf("expected 8 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.TotalTokens != 20 {
		t.Errorf("expected 20 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestGeminiProvider_Complete_WithSystem(t *testing.T) {
	var receivedBody geminiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		resp := geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []geminiPart `json:"parts"`
				} `json:"content"`
			}{
				{Content: struct {
					Parts []geminiPart `json:"parts"`
				}{Parts: []geminiPart{{Text: "ok"}}}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.System = "You are a Gemini assistant"
	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if receivedBody.SystemInstruction == nil {
		t.Fatal("expected system instruction in request")
	}
	if len(receivedBody.SystemInstruction.Parts) == 0 || receivedBody.SystemInstruction.Parts[0].Text != "You are a Gemini assistant" {
		t.Errorf("expected system instruction text, got %v", receivedBody.SystemInstruction)
	}
}

func TestGeminiProvider_Complete_WithModelOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the model appears in the URL path
		if !strings.Contains(r.URL.Path, "gemini-flash") {
			t.Errorf("expected model 'gemini-flash' in URL path, got: %s", r.URL.Path)
		}
		resp := geminiResponse{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.Model = "gemini-flash"
	// resp may have empty content; just ensure no error
	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
}

func TestGeminiProvider_Complete_NoCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("expected empty content for no candidates, got '%s'", resp.Content)
	}
}

func TestGeminiProvider_Complete_NoUsageMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []geminiPart `json:"parts"`
				} `json:"content"`
			}{
				{Content: struct {
					Parts []geminiPart `json:"parts"`
				}{Parts: []geminiPart{{Text: "response"}}}},
			},
			UsageMetadata: nil, // no usage info
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	// Usage should default to zero when not present
	if resp.Usage.TotalTokens != 0 {
		t.Errorf("expected 0 total tokens when no usage metadata, got %d", resp.Usage.TotalTokens)
	}
}

func TestGeminiProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"code": 400, "message": "bad request"}}`)) //nolint:errcheck
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "bad-key",
		defaultModel: "gemini-pro",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for API error status")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to contain status code, got: %v", err)
	}
}

func TestGeminiProvider_Complete_ValidationError(t *testing.T) {
	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       &http.Client{},
	}
	req := &Request{
		Prompt:      "Hello",
		MaxTokens:   100,
		Temperature: -0.5, // invalid
	}
	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGeminiProvider_Stream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		chunk1 := geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []geminiPart `json:"parts"`
				} `json:"content"`
			}{
				{Content: struct {
					Parts []geminiPart `json:"parts"`
				}{Parts: []geminiPart{{Text: "Hello"}}}},
			},
		}
		chunk2 := geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []geminiPart `json:"parts"`
				} `json:"content"`
			}{
				{Content: struct {
					Parts []geminiPart `json:"parts"`
				}{Parts: []geminiPart{{Text: ", Gemini!"}}}},
			},
		}

		b1, _ := json.Marshal(chunk1)
		b2, _ := json.Marshal(chunk2)
		fmt.Fprintf(w, "data: %s\n\n", b1)
		fmt.Fprintf(w, "data: %s\n\n", b2)
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if buf.String() != "Hello, Gemini!" {
		t.Errorf("expected 'Hello, Gemini!', got '%s'", buf.String())
	}
}

func TestGeminiProvider_Stream_WithSystem(t *testing.T) {
	var receivedBody geminiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		w.Header().Set("Content-Type", "text/event-stream")
		// empty stream
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
		client:       newTestClient(server.URL),
	}

	req := NewRequest("Hello")
	req.System = "Be brief"
	var buf bytes.Buffer
	err := p.Stream(context.Background(), req, &buf)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if receivedBody.SystemInstruction == nil {
		t.Error("expected system instruction in stream request")
	}
}

func TestGeminiProvider_Stream_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`)) //nolint:errcheck
	}))
	defer server.Close()

	p := &GeminiProvider{
		apiKey:       "bad-key",
		defaultModel: "gemini-pro",
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

func TestGeminiProvider_Stream_ValidationError(t *testing.T) {
	p := &GeminiProvider{
		apiKey:       "test-key",
		defaultModel: "gemini-pro",
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

func TestGeminiProvider_Init_EmptyAPIKey(t *testing.T) {
	t.Cleanup(resetRegistry)

	Register("gemini-init-test", func(apiKey string) (Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("API key required for Gemini provider")
		}
		return &GeminiProvider{
			apiKey:       apiKey,
			defaultModel: "gemini-pro",
			client:       &http.Client{},
		}, nil
	})

	_, err := GetProvider("gemini-init-test", "")
	if err == nil {
		t.Error("expected error for empty API key")
	}
}
