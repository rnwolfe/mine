package ai

import (
	"testing"
)

func TestNewRequest(t *testing.T) {
	req := NewRequest("test prompt")

	if req.Prompt != "test prompt" {
		t.Errorf("expected prompt 'test prompt', got '%s'", req.Prompt)
	}

	if req.MaxTokens != 4096 {
		t.Errorf("expected default MaxTokens 4096, got %d", req.MaxTokens)
	}

	if req.Temperature != 0.7 {
		t.Errorf("expected default Temperature 0.7, got %f", req.Temperature)
	}
}

func TestRequestCustomization(t *testing.T) {
	req := NewRequest("test")
	req.System = "You are a helpful assistant"
	req.Model = "custom-model"
	req.MaxTokens = 1000
	req.Temperature = 0.5

	if req.System != "You are a helpful assistant" {
		t.Errorf("system message not set correctly")
	}

	if req.Model != "custom-model" {
		t.Errorf("model not set correctly")
	}

	if req.MaxTokens != 1000 {
		t.Errorf("MaxTokens not set correctly")
	}

	if req.Temperature != 0.5 {
		t.Errorf("Temperature not set correctly")
	}
}

func TestRequestValidation(t *testing.T) {
	tests := []struct {
		name        string
		req         *Request
		expectError bool
	}{
		{
			name: "valid request with defaults",
			req:  NewRequest("test"),
			expectError: false,
		},
		{
			name: "valid request with temperature 0.0",
			req: &Request{
				Prompt:      "test",
				MaxTokens:   100,
				Temperature: 0.0,
			},
			expectError: false,
		},
		{
			name: "valid request with temperature 1.0",
			req: &Request{
				Prompt:      "test",
				MaxTokens:   100,
				Temperature: 1.0,
			},
			expectError: false,
		},
		{
			name: "invalid temperature below 0",
			req: &Request{
				Prompt:      "test",
				MaxTokens:   100,
				Temperature: -0.1,
			},
			expectError: true,
		},
		{
			name: "invalid temperature above 1.0",
			req: &Request{
				Prompt:      "test",
				MaxTokens:   100,
				Temperature: 1.5,
			},
			expectError: true,
		},
		{
			name: "invalid max_tokens zero",
			req: &Request{
				Prompt:      "test",
				MaxTokens:   0,
				Temperature: 0.7,
			},
			expectError: true,
		},
		{
			name: "invalid max_tokens negative",
			req: &Request{
				Prompt:      "test",
				MaxTokens:   -100,
				Temperature: 0.7,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.expectError && err == nil {
				t.Errorf("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no validation error, got: %v", err)
			}
		})
	}
}
