package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	claudeAPIURL     = "https://api.anthropic.com/v1/messages"
	claudeAPIVersion = "2023-06-01"
)

// ClaudeProvider implements the Provider interface for Anthropic's Claude API.
type ClaudeProvider struct {
	config ProviderConfig
	client *http.Client
}

// NewClaudeProvider creates a new Claude provider.
func NewClaudeProvider(config ProviderConfig) *ClaudeProvider {
	if config.BaseURL == "" {
		config.BaseURL = claudeAPIURL
	}
	if config.Model == "" {
		config.Model = "claude-sonnet-4-5-20250929"
	}

	return &ClaudeProvider{
		config: config,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name returns the provider name.
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// Complete sends a non-streaming request.
func (p *ClaudeProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	reqBody := p.buildRequest(req, false)
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return p.parseResponse(&result)
}

// Stream sends a streaming request.
func (p *ClaudeProvider) Stream(ctx context.Context, req CompletionRequest, callback StreamCallback) (*CompletionResponse, error) {
	reqBody := p.buildRequest(req, true)
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return p.processStream(resp.Body, callback)
}

// buildRequest constructs the Claude API request body.
func (p *ClaudeProvider) buildRequest(req CompletionRequest, stream bool) map[string]interface{} {
	messages := make([]map[string]string, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	temp := req.Temperature
	if temp == 0 {
		temp = 1.0
	}

	return map[string]interface{}{
		"model":       p.config.Model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temp,
		"stream":      stream,
	}
}

// setHeaders sets required headers for Claude API.
func (p *ClaudeProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)
}

// processStream handles SSE streaming from Claude API.
func (p *ClaudeProvider) processStream(body io.Reader, callback StreamCallback) (*CompletionResponse, error) {
	scanner := bufio.NewScanner(body)
	var fullContent strings.Builder
	var usage Usage

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event claudeStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue // Skip malformed events
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta.Text != "" {
				fullContent.WriteString(event.Delta.Text)
				if callback != nil {
					if err := callback(event.Delta.Text); err != nil {
						return nil, err
					}
				}
			}
		case "message_delta":
			if event.Usage.OutputTokens > 0 {
				usage.OutputTokens = event.Usage.OutputTokens
			}
		case "message_start":
			if event.Message.Usage.InputTokens > 0 {
				usage.InputTokens = event.Message.Usage.InputTokens
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream error: %w", err)
	}

	return &CompletionResponse{
		Content: fullContent.String(),
		Usage:   usage,
	}, nil
}

// parseResponse converts Claude API response to our format.
func (p *ClaudeProvider) parseResponse(resp *claudeResponse) (*CompletionResponse, error) {
	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	return &CompletionResponse{
		Content: resp.Content[0].Text,
		Usage: Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}, nil
}

// Claude API response structures.
type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type claudeStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Message struct {
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
