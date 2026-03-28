package triton

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL      string
	modelName    string
	maxNewTokens int
	httpClient   *http.Client
}

type inferRequest struct {
	Inputs []inputTensor `json:"inputs"`
}

type inputTensor struct {
	Name     string      `json:"name"`
	Shape    []int       `json:"shape"`
	Datatype string      `json:"datatype"`
	Data     interface{} `json:"data"`
}

type inferResponse struct {
	Outputs []outputTensor `json:"outputs"`
}

type outputTensor struct {
	Name  string        `json:"name"`
	Data  []interface{} `json:"data"`
	Shape []int         `json:"shape,omitempty"`
}

func NewClient(baseURL, modelName string, timeout time.Duration, maxNewTokens int) *Client {
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		modelName:    modelName,
		maxNewTokens: maxNewTokens,
		httpClient:   &http.Client{Timeout: timeout},
	}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.getHealth(ctx, c.baseURL+"/v2/health/ready"); err != nil {
		return err
	}
	modelReadyURL := fmt.Sprintf("%s/v2/models/%s/ready", c.baseURL, url.PathEscape(c.modelName))
	return c.getHealth(ctx, modelReadyURL)
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	payload, err := json.Marshal(buildInferRequest(prompt, c.maxNewTokens))
	if err != nil {
		return "", err
	}

	inferURL := fmt.Sprintf("%s/v2/models/%s/infer", c.baseURL, url.PathEscape(c.modelName))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inferURL, bytes.NewReader(payload))
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
		return "", fmt.Errorf("triton infer status %d", resp.StatusCode)
	}

	var inferResp inferResponse
	if err := json.NewDecoder(resp.Body).Decode(&inferResp); err != nil {
		return "", err
	}

	return extractGeneratedText(inferResp)
}

func buildInferRequest(prompt string, maxNewTokens int) inferRequest {
	return inferRequest{
		Inputs: []inputTensor{
			{
				Name:     "prompt",
				Shape:    []int{1},
				Datatype: "BYTES",
				Data:     []string{prompt},
			},
			{
				Name:     "max_new_tokens",
				Shape:    []int{1},
				Datatype: "INT32",
				Data:     []int{maxNewTokens},
			},
		},
	}
}

func extractGeneratedText(resp inferResponse) (string, error) {
	for _, output := range resp.Outputs {
		if output.Name != "generated_text" || len(output.Data) == 0 {
			continue
		}
		value := output.Data[0]
		switch text := value.(type) {
		case string:
			if strings.TrimSpace(text) == "" {
				return "", fmt.Errorf("empty generated_text output")
			}
			return text, nil
		case map[string]interface{}:
			if nested, ok := text["b64"].(string); ok && nested != "" {
				return nested, nil
			}
		}
	}
	return "", fmt.Errorf("generated_text output missing")
}

func (c *Client) getHealth(ctx context.Context, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("triton health status %d", resp.StatusCode)
	}
	return nil
}
