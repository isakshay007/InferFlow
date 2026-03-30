package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthReturnsNilWhenServerReturns200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "qwen2.5:0.5b", 2*time.Second)
	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestHealthReturnsErrorWhenServerReturns503(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(server.URL, "qwen2.5:0.5b", 2*time.Second)
	if err := client.Health(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGenerateReturnsResponseTextOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if in["model"] != "qwen2.5:0.5b" {
			t.Fatalf("unexpected model: %v", in["model"])
		}
		if in["stream"] != false {
			t.Fatalf("expected stream false, got %v", in["stream"])
		}
		prompt, _ := in["prompt"].(string)
		if !strings.Contains(prompt, "System: You are concise.") {
			t.Fatalf("unexpected prompt: %q", prompt)
		}
		if !strings.Contains(prompt, "User: Say hello") {
			t.Fatalf("unexpected prompt: %q", prompt)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"response": "hello from ollama"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "qwen2.5:0.5b", 2*time.Second)
	got, err := client.Generate(context.Background(), []map[string]string{
		{"role": "system", "content": "You are concise."},
		{"role": "user", "content": "Say hello"},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != "hello from ollama" {
		t.Fatalf("unexpected response: %q", got)
	}
}

func TestGenerateReturnsErrorWhenServerReturns500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "qwen2.5:0.5b", 2*time.Second)
	if _, err := client.Generate(context.Background(), []map[string]string{
		{"role": "user", "content": "hello"},
	}); err == nil {
		t.Fatal("expected error, got nil")
	}
}
