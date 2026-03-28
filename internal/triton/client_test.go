package triton

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildInferRequest(t *testing.T) {
	req := buildInferRequest("hello", 64)

	if len(req.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(req.Inputs))
	}
	if req.Inputs[0].Name != "prompt" {
		t.Fatalf("expected prompt input, got %s", req.Inputs[0].Name)
	}
	if req.Inputs[1].Name != "max_new_tokens" {
		t.Fatalf("expected max_new_tokens input, got %s", req.Inputs[1].Name)
	}
}

func TestExtractGeneratedText(t *testing.T) {
	text, err := extractGeneratedText(inferResponse{
		Outputs: []outputTensor{{
			Name: "generated_text",
			Data: []interface{}{"hello from triton"},
		}},
	})
	if err != nil {
		t.Fatalf("extract generated text: %v", err)
	}
	if text != "hello from triton" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestExtractGeneratedTextErrorsOnMissingOutput(t *testing.T) {
	_, err := extractGeneratedText(inferResponse{})
	if err == nil {
		t.Fatal("expected error for missing generated_text")
	}
}

func TestClientHealthCheckAndGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/health/ready":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/v2/models/qwen3_0_6b/ready":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/v2/models/qwen3_0_6b/infer":
			var req inferRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode infer request: %v", err)
			}
			if len(req.Inputs) != 2 {
				t.Fatalf("expected 2 inputs, got %d", len(req.Inputs))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(inferResponse{
				Outputs: []outputTensor{{
					Name: "generated_text",
					Data: []interface{}{"adapter output"},
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "qwen3_0_6b", 2*time.Second, 64)

	if err := client.HealthCheck(context.Background()); err != nil {
		t.Fatalf("health check: %v", err)
	}

	got, err := client.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got != "adapter output" {
		t.Fatalf("unexpected generated text: %q", got)
	}
}

func TestClientGenerateHandlesTritonError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/infer"):
			http.Error(w, "bad gateway", http.StatusBadGateway)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "qwen3_0_6b", 2*time.Second, 64)
	if _, err := client.Generate(context.Background(), "hello"); err == nil {
		t.Fatal("expected generate error")
	}
}
