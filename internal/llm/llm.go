package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Provider abstracts LLM backends.
type Provider interface {
	Name() string
	Chat(ctx context.Context, messages []Message) (string, error)
}

// Config configures an LLM provider.
type Config struct {
	Type    string // ollama | openai
	BaseURL string
	Model   string
	APIKey  string
}

// New creates an LLM provider from config.
func New(cfg Config) (Provider, error) {
	switch cfg.Type {
	case "ollama":
		return &Ollama{baseURL: cfg.BaseURL, model: cfg.Model}, nil
	case "openai":
		return &OpenAICompatible{baseURL: cfg.BaseURL, model: cfg.Model, apiKey: cfg.APIKey}, nil
	default:
		return nil, fmt.Errorf("unknown LLM provider type: %s", cfg.Type)
	}
}

// Ollama implements the Ollama local LLM API.
type Ollama struct {
	baseURL string
	model   string
	client  *http.Client
}

func (o *Ollama) Name() string { return "ollama" }

func (o *Ollama) Chat(ctx context.Context, messages []Message) (string, error) {
	url := o.baseURL + "/api/chat"
	if o.baseURL == "" {
		url = "http://localhost:11434/api/chat"
	}
	body := map[string]any{
		"model":    o.model,
		"messages": messages,
		"stream":   false,
		"format":   "json",
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := o.client
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	return result.Message.Content, nil
}

// OpenAICompatible implements OpenAI-compatible chat APIs.
type OpenAICompatible struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
}

func (o *OpenAICompatible) Name() string { return "openai" }

func (o *OpenAICompatible) Chat(ctx context.Context, messages []Message) (string, error) {
	url := o.baseURL + "/v1/chat/completions"
	if o.baseURL == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}
	body := map[string]any{
		"model":    o.model,
		"messages": messages,
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	client := o.client
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}
	return result.Choices[0].Message.Content, nil
}
