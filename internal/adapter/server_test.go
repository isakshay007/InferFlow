package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"inferflow/internal/proxy"
)

type fakeGenerator struct {
	healthErr error
	output    string
	genErr    error
	prompt    string
}

func (f *fakeGenerator) HealthCheck(context.Context) error {
	return f.healthErr
}

func (f *fakeGenerator) Generate(_ context.Context, prompt string) (string, error) {
	f.prompt = prompt
	if f.genErr != nil {
		return "", f.genErr
	}
	return f.output, nil
}

func TestHealthz(t *testing.T) {
	handler := NewHandler(&fakeGenerator{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestInferTranslatesPrompt(t *testing.T) {
	client := &fakeGenerator{output: "hello from qwen"}
	handler := NewHandler(client)

	body := proxy.BackendRequest{
		Model: "qwen3-0.6b",
		Messages: []proxy.ChatMessage{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Say hello"},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/infer", mustJSON(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if client.prompt != "system: You are helpful.\nuser: Say hello" {
		t.Fatalf("unexpected prompt: %q", client.prompt)
	}

	var resp proxy.BackendResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.OutputText != "hello from qwen" {
		t.Fatalf("unexpected output: %q", resp.OutputText)
	}
}

func TestInferHandlesGeneratorFailure(t *testing.T) {
	handler := NewHandler(&fakeGenerator{genErr: errors.New("boom")})
	req := httptest.NewRequest(http.MethodPost, "/infer", mustJSON(proxy.BackendRequest{
		Model: "qwen3-0.6b",
		Messages: []proxy.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

func mustJSON(v any) *bytes.Buffer {
	data, _ := json.Marshal(v)
	return bytes.NewBuffer(data)
}
