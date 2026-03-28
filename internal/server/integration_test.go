package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"inferflow/internal/proxy"
	"inferflow/internal/router"
)

func TestEndToEndChatCompletion(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "/infer":
			writeJSON(w, http.StatusOK, map[string]string{
				"model":       "mock-llm",
				"output_text": "Mock response to: integration prompt",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()

	srv := newTestServer(t, []string{backend.URL})
	defer func() { _ = srv.Shutdown() }()

	app := httptest.NewServer(srv.httpSrv.Handler)
	defer app.Close()

	reqBody := proxy.ChatCompletionRequest{
		Model: "mock-llm",
		Messages: []proxy.ChatMessage{{
			Role:    "user",
			Content: "integration prompt",
		}},
	}
	data, _ := json.Marshal(reqBody)

	resp, err := http.Post(app.URL+"/v1/chat/completions", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestEndToEndSwitchStrategyThenChatCompletion(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "/infer":
			writeJSON(w, http.StatusOK, map[string]string{
				"model":       "mock-llm",
				"output_text": "Mock response to: switched strategy prompt",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()

	srv := newTestServer(t, []string{backend.URL})
	defer func() { _ = srv.Shutdown() }()

	app := httptest.NewServer(srv.httpSrv.Handler)
	defer app.Close()

	switchReqBody := []byte(`{"strategy":"least_pending"}`)
	switchReq, err := http.NewRequest(http.MethodPut, app.URL+"/strategy", bytes.NewReader(switchReqBody))
	if err != nil {
		t.Fatalf("build switch strategy request: %v", err)
	}
	switchReq.Header.Set("Content-Type", "application/json")

	switchResp, err := http.DefaultClient.Do(switchReq)
	if err != nil {
		t.Fatalf("switch strategy request: %v", err)
	}
	defer switchResp.Body.Close()

	if switchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected strategy switch status 200, got %d", switchResp.StatusCode)
	}

	var strategyResp map[string]string
	if err := json.NewDecoder(switchResp.Body).Decode(&strategyResp); err != nil {
		t.Fatalf("decode strategy response: %v", err)
	}
	if strategyResp["strategy"] != router.StrategyLeastPending {
		t.Fatalf("expected strategy %q, got %q", router.StrategyLeastPending, strategyResp["strategy"])
	}

	reqBody := proxy.ChatCompletionRequest{
		Model: "mock-llm",
		Messages: []proxy.ChatMessage{{
			Role:    "user",
			Content: "switched strategy prompt",
		}},
	}
	data, _ := json.Marshal(reqBody)

	resp, err := http.Post(app.URL+"/v1/chat/completions", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post chat completion: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected chat completion status 200, got %d", resp.StatusCode)
	}
}
