package ollama

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
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewClient(baseURL, model string, timeout time.Duration) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama health status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Generate(ctx context.Context, messages []map[string]string) (string, error) {
	payload := map[string]any{
		"model":  c.model,
		"prompt": buildPrompt(messages),
		"stream": false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama generate status %d", resp.StatusCode)
	}

	var out struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Response, nil
}

func buildPrompt(messages []map[string]string) string {
	var parts []string
	for _, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg["role"]))
		content := strings.TrimSpace(msg["content"])
		if content == "" {
			continue
		}

		prefix := "User"
		switch role {
		case "system":
			prefix = "System"
		case "assistant":
			prefix = "Assistant"
		case "user":
			prefix = "User"
		}

		parts = append(parts, fmt.Sprintf("%s: %s", prefix, content))
	}
	return strings.Join(parts, "\n")
}
