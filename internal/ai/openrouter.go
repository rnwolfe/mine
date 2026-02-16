package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
)

// OpenRouterProvider implements the Provider interface for OpenRouter.
type OpenRouterProvider struct {
	apiKey       string
	defaultModel string
	client       *http.Client
}

func init() {
	Register("openrouter", func(apiKey string) (Provider, error) {
		// Free models don't require an API key, use a placeholder
		if apiKey == "" {
			apiKey = "sk-or-v1-free" // Placeholder for free models
		}
		return &OpenRouterProvider{
			apiKey:       apiKey,
			defaultModel: "z-ai/glm-4.5-air:free", // Free model fallback
			client:       &http.Client{},
		}, nil
	})
}

func (o *OpenRouterProvider) Name() string {
	return "openrouter"
}

func (o *OpenRouterProvider) Complete(ctx context.Context, req *Request) (*Response, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	model := req.Model
	if model == "" {
		model = o.defaultModel
	}

	messages := []openRouterMessage{
		{Role: "user", Content: req.Prompt},
	}

	if req.System != "" {
		// Prepend system message
		messages = append([]openRouterMessage{
			{Role: "system", Content: req.System},
		}, messages...)
	}

	apiReq := openRouterRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openRouterAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/rnwolfe/mine") // Required by OpenRouter
	httpReq.Header.Set("X-Title", "mine CLI")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenRouter API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	content := ""
	if len(apiResp.Choices) > 0 {
		content = apiResp.Choices[0].Message.Content
	}

	return &Response{
		Content: content,
		Model:   apiResp.Model,
		Usage: Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
	}, nil
}

func (o *OpenRouterProvider) Stream(ctx context.Context, req *Request, w io.Writer) error {
	if err := req.Validate(); err != nil {
		return err
	}

	model := req.Model
	if model == "" {
		model = o.defaultModel
	}

	messages := []openRouterMessage{
		{Role: "user", Content: req.Prompt},
	}

	if req.System != "" {
		messages = append([]openRouterMessage{
			{Role: "system", Content: req.System},
		}, messages...)
	}

	apiReq := openRouterRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openRouterAPIURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/rnwolfe/mine")
	httpReq.Header.Set("X-Title", "mine CLI")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenRouter API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream line-by-line
	var buffer []byte
	buf := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			buffer = append(buffer, buf[:n]...)

			// Process complete lines
			for {
				idx := bytes.IndexByte(buffer, '\n')
				if idx == -1 {
					break
				}

				line := string(bytes.TrimSpace(buffer[:idx]))
				buffer = buffer[idx+1:]

				// SSE data lines start with "data: "
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")

					// Skip special SSE messages
					if data == "[DONE]" {
						return nil
					}

					var event openRouterStreamEvent
					if err := json.Unmarshal([]byte(data), &event); err == nil {
						if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
							if _, err := w.Write([]byte(event.Choices[0].Delta.Content)); err != nil {
								return err
							}
						}
					}
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Process any remaining partial line in the buffer after EOF
	if len(buffer) > 0 {
		line := string(bytes.TrimSpace(buffer))
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "[DONE]" {
				var event openRouterStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
						if _, err := w.Write([]byte(event.Choices[0].Delta.Content)); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

// OpenRouter API types (OpenAI-compatible)
type openRouterRequest struct {
	Model       string               `json:"model"`
	Messages    []openRouterMessage  `json:"messages"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
	Temperature float64              `json:"temperature,omitempty"`
	Stream      bool                 `json:"stream,omitempty"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openRouterStreamEvent struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}
