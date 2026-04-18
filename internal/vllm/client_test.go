package vllm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientHealthCheckAndGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/v1/completions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"text":"hello from vllm"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "Qwen/Qwen2.5-0.5B-Instruct", 2*time.Second, 64, 0)
	if err := client.HealthCheck(context.Background()); err != nil {
		t.Fatalf("health check: %v", err)
	}

	text, err := client.Generate(context.Background(), "user: hello")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if text != "hello from vllm" {
		t.Fatalf("expected completion text, got %q", text)
	}
}
