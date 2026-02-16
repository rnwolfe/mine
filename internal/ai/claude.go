package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	claudeAPIURL     = "https://api.anthropic.com/v1/messages"
	claudeAPIVersion = "2023-06-01"
)

// ClaudeProvider implements the Provider interface for Anthropic's Claude.
type ClaudeProvider struct {
	apiKey       string
	defaultModel string
	client       *http.Client
}

func init() {
	Register("claude", func(apiKey string) (Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("API key required for Claude provider")
		}
		return &ClaudeProvider{
			apiKey:       apiKey,
			defaultModel: "claude-sonnet-4-5-20250929",
			client: &http.Client{
				Timeout: 60 * time.Second,
			},
		}, nil
	})
}

func (c *ClaudeProvider) Name() string {
	return "claude"
}

func (c *ClaudeProvider) Complete(ctx context.Context, req *Request) (*Response, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}

	apiReq := claudeRequest{
		Model:     model,
		MaxTokens: req.MaxTokens,
		Messages: []claudeMessage{
			{Role: "user", Content: req.Prompt},
		},
	}

	if req.System != "" {
		apiReq.System = req.System
	}

	if req.Temperature > 0 {
		apiReq.Temperature = req.Temperature
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	content := ""
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &Response{
		Content: content,
		Model:   apiResp.Model,
		Usage: Usage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
	}, nil
}

func (c *ClaudeProvider) Stream(ctx context.Context, req *Request, w io.Writer) error {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}

	apiReq := claudeRequest{
		Model:     model,
		MaxTokens: req.MaxTokens,
		Messages: []claudeMessage{
			{Role: "user", Content: req.Prompt},
		},
		Stream: true,
	}

	if req.System != "" {
		apiReq.System = req.System
	}

	if req.Temperature > 0 {
		apiReq.Temperature = req.Temperature
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream (simplified - real SSE has "data: " prefix and event types)
	buf := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}

		// Parse event data
		chunk := string(buf[:n])
		var event claudeStreamEvent
		if err := json.Unmarshal([]byte(chunk), &event); err == nil {
			if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
				if _, err := w.Write([]byte(event.Delta.Text)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Claude API types
type claudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []claudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Model   string `json:"model"`
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
}
