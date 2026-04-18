package vllm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL      string
	modelName    string
	maxTokens    int
	temperature  float64
	httpClient   *http.Client
}

type completionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type completionResponse struct {
	Choices []completionChoice `json:"choices"`
}

type completionChoice struct {
	Text string `json:"text"`
}

func NewClient(baseURL, modelName string, timeout time.Duration, maxTokens int, temperature float64) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		modelName:   modelName,
		maxTokens:   maxTokens,
		temperature: temperature,
		httpClient:  &http.Client{Timeout: timeout},
	}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("vllm health status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	payload, err := json.Marshal(completionRequest{
		Model:       c.modelName,
		Prompt:      prompt,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("vllm completion status %d", resp.StatusCode)
	}

	var completion completionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return "", err
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Text) == "" {
		return "", fmt.Errorf("vllm completion returned no text")
	}
	return strings.TrimSpace(completion.Choices[0].Text), nil
}
