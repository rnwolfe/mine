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
