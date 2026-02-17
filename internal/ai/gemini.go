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
	geminiAPIBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// GeminiProvider implements the Provider interface for Google's Gemini.
type GeminiProvider struct {
	apiKey       string
	defaultModel string
	client       *http.Client
}

func init() {
	Register("gemini", func(apiKey string) (Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("API key required for Gemini provider")
		}
		return &GeminiProvider{
			apiKey:       apiKey,
			defaultModel: "gemini-2.0-flash-exp",
			client:       &http.Client{},
		}, nil
	})
}

func (g *GeminiProvider) Name() string {
	return "gemini"
}

func (g *GeminiProvider) Complete(ctx context.Context, req *Request) (*Response, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	model := req.Model
	if model == "" {
		model = g.defaultModel
	}

	// Build the content parts
	parts := []geminiPart{
		{Text: req.Prompt},
	}

	contents := []geminiContent{
		{
			Role:  "user",
			Parts: parts,
		},
	}

	// Prepend system instruction if provided
	apiReq := geminiRequest{
		Contents: contents,
		GenerationConfig: geminiGenerationConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		},
	}

	if req.System != "" {
		apiReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{
				{Text: req.System},
			},
		}
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, err
	}

	// SECURITY NOTE: Google's Gemini API requires the API key as a URL query parameter.
	// This is the official authentication method per Google's documentation:
	// https://ai.google.dev/gemini-api/docs/api-key
	//
	// Known limitation: URL query parameters can be logged by proxies, servers, and browsers.
	// Google does not currently support header-based authentication for the Gemini API.
	// Users should be aware that API keys may appear in server logs.
	//
	// Mitigation: API keys are stored encrypted at rest in the keystore and only transmitted
	// over HTTPS. For production use, consider using Google Cloud's Vertex AI API which
	// supports service account authentication.
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		geminiAPIBaseURL, model, g.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	content := ""
	if len(apiResp.Candidates) > 0 && len(apiResp.Candidates[0].Content.Parts) > 0 {
		content = apiResp.Candidates[0].Content.Parts[0].Text
	}

	usage := Usage{}
	if apiResp.UsageMetadata != nil {
		usage.PromptTokens = apiResp.UsageMetadata.PromptTokenCount
		usage.CompletionTokens = apiResp.UsageMetadata.CandidatesTokenCount
		usage.TotalTokens = apiResp.UsageMetadata.TotalTokenCount
	}

	return &Response{
		Content: content,
		Model:   model,
		Usage:   usage,
	}, nil
}

func (g *GeminiProvider) Stream(ctx context.Context, req *Request, w io.Writer) error {
	if err := req.Validate(); err != nil {
		return err
	}

	model := req.Model
	if model == "" {
		model = g.defaultModel
	}

	parts := []geminiPart{
		{Text: req.Prompt},
	}

	contents := []geminiContent{
		{
			Role:  "user",
			Parts: parts,
		},
	}

	apiReq := geminiRequest{
		Contents: contents,
		GenerationConfig: geminiGenerationConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		},
	}

	if req.System != "" {
		apiReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{
				{Text: req.System},
			},
		}
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return err
	}

	// SECURITY NOTE: Google's Gemini API requires the API key as a URL query parameter.
	// See comment in Complete() method for full details on this limitation.
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse",
		geminiAPIBaseURL, model, g.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream line-by-line
	var buffer []byte
	buf := make([]byte, 4096)

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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

					var event geminiResponse
					if err := json.Unmarshal([]byte(data), &event); err == nil {
						if len(event.Candidates) > 0 && len(event.Candidates[0].Content.Parts) > 0 {
							text := event.Candidates[0].Content.Parts[0].Text
							if text != "" {
								if _, err := w.Write([]byte(text)); err != nil {
									return err
								}
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
			var event geminiResponse
			if err := json.Unmarshal([]byte(data), &event); err == nil {
				if len(event.Candidates) > 0 && len(event.Candidates[0].Content.Parts) > 0 {
					text := event.Candidates[0].Content.Parts[0].Text
					if text != "" {
						if _, err := w.Write([]byte(text)); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

// Gemini API types
type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata,omitempty"`
}
