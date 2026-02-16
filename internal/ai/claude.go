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
	defaultModel     = "claude-sonnet-4-5-20250929"
	defaultMaxTokens = 4096
	requestTimeout   = 120 * time.Second
)

// ClaudeProvider implements the Provider interface for Anthropic's Claude API.
type ClaudeProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewClaudeProvider creates a new Claude provider.
func NewClaudeProvider(apiKey, model string) *ClaudeProvider {
	if model == "" {
		model = defaultModel
	}
	return &ClaudeProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (c *ClaudeProvider) Name() string {
	return "claude"
}

func (c *ClaudeProvider) Complete(ctx context.Context, req *Request) (*Response, error) {
	apiReq := c.buildRequest(req, false)
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, &ProviderError{Provider: "claude", Message: "encoding request", Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, &ProviderError{Provider: "claude", Message: "creating request", Err: err}
	}
	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: "claude", Message: "sending request", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var apiResp claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, &ProviderError{Provider: "claude", Message: "decoding response", Err: err}
	}

	return c.parseResponse(&apiResp), nil
}

func (c *ClaudeProvider) Stream(ctx context.Context, req *Request, w io.Writer) error {
	apiReq := c.buildRequest(req, true)
	body, err := json.Marshal(apiReq)
	if err != nil {
		return &ProviderError{Provider: "claude", Message: "encoding request", Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return &ProviderError{Provider: "claude", Message: "creating request", Err: err}
	}
	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return &ProviderError{Provider: "claude", Message: "sending request", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	return c.streamResponse(resp.Body, w)
}

func (c *ClaudeProvider) buildRequest(req *Request, stream bool) map[string]interface{} {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}

	messages := []map[string]string{
		{"role": "user", "content": req.Prompt},
	}

	apiReq := map[string]interface{}{
		"model":      c.model,
		"max_tokens": maxTokens,
		"messages":   messages,
		"stream":     stream,
	}

	if req.System != "" {
		apiReq["system"] = req.System
	}

	if req.Temperature > 0 {
		apiReq["temperature"] = req.Temperature
	}

	return apiReq
}

func (c *ClaudeProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)
}

func (c *ClaudeProvider) streamResponse(body io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(body)
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

		// Write content deltas to output
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			if _, err := w.Write([]byte(event.Delta.Text)); err != nil {
				return &ProviderError{Provider: "claude", Message: "writing output", Err: err}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return &ProviderError{Provider: "claude", Message: "reading stream", Err: err}
	}
	return nil
}

func (c *ClaudeProvider) parseResponse(apiResp *claudeResponse) *Response {
	var content string
	if len(apiResp.Content) > 0 {
		content = apiResp.Content[0].Text
	}

	return &Response{
		Content: content,
		Model:   apiResp.Model,
		Usage: &Usage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
		},
	}
}

func (c *ClaudeProvider) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ProviderError{
			Provider: "claude",
			Message:  fmt.Sprintf("reading error response (status %d)", resp.StatusCode),
			Err:      err,
		}
	}

	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		return &ProviderError{
			Provider: "claude",
			Message:  fmt.Sprintf("API error (status %d)", resp.StatusCode),
		}
	}
	return &ProviderError{
		Provider: "claude",
		Message:  fmt.Sprintf("%s (status %d)", errResp.Error.Message, resp.StatusCode),
	}
}

// claudeResponse is the API response structure from Claude.
type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// claudeStreamEvent represents a single event in the streaming response.
type claudeStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}
