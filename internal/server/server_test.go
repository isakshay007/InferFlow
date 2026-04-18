package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"inferflow/internal/proxy"
	"inferflow/internal/router"
)

func TestChatCompletionsSuccess(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "/infer":
			writeJSON(w, http.StatusOK, map[string]string{
				"model":       "mock-llm",
				"output_text": "Mock response to: Hello",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()

	srv := newTestServer(t, []string{backend.URL})
	defer func() { _ = srv.Shutdown() }()

	body := proxy.ChatCompletionRequest{
		Model: "mock-llm",
		Messages: []proxy.ChatMessage{{
			Role:    "user",
			Content: "Hello",
		}},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", mustJSON(body))
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp proxy.ChatCompletionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content == "" {
		t.Fatalf("expected one non-empty choice, got %+v", resp.Choices)
	}
}

func TestChatCompletionsRejectsMalformedJSON(t *testing.T) {
	srv := newTestServer(t, []string{"http://127.0.0.1:1"})
	defer func() { _ = srv.Shutdown() }()

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestReadyzReflectsBackendHealth(t *testing.T) {
	backend, _ := router.NewBackend("a", "http://backend.test")
	backend.SetHealthy(false)

	srv, err := New(Config{
		ListenAddr:           ":0",
		Backends:             []*router.Backend{backend},
		ProbeInterval:        10 * time.Hour,
		BackendRequestTimout: time.Second,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer func() { _ = srv.Shutdown() }()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
}

func TestChatCompletionsReturnsBadGatewayWhenBackendFails(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "/infer":
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()

	srv := newTestServer(t, []string{backend.URL})
	defer func() { _ = srv.Shutdown() }()

	body := proxy.ChatCompletionRequest{
		Model: "mock-llm",
		Messages: []proxy.ChatMessage{{
			Role:    "user",
			Content: "Hello",
		}},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", mustJSON(body))
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}
}

func TestStrategyEndpointDefaultsToRoundRobin(t *testing.T) {
	srv := newTestServer(t, []string{"http://127.0.0.1:1"})
	defer func() { _ = srv.Shutdown() }()

	req := httptest.NewRequest(http.MethodGet, "/strategy", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["strategy"] != router.StrategyRoundRobin {
		t.Fatalf("expected strategy %q, got %q", router.StrategyRoundRobin, resp["strategy"])
	}
}

func TestStrategyEndpointSupportsSwitching(t *testing.T) {
	srv := newTestServer(t, []string{"http://127.0.0.1:1"})
	defer func() { _ = srv.Shutdown() }()

	req := httptest.NewRequest(http.MethodPut, "/strategy", bytes.NewBufferString(`{"strategy":"least-pending"}`))
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/strategy", nil)
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["strategy"] != router.StrategyLeastPending {
		t.Fatalf("expected strategy %q, got %q", router.StrategyLeastPending, resp["strategy"])
	}

	req = httptest.NewRequest(http.MethodPut, "/strategy", bytes.NewBufferString(`{"strategy":"random"}`))
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/strategy", bytes.NewBufferString(`{"strategy":"kv_aware"}`))
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestStrategyEndpointRejectsInvalidStrategy(t *testing.T) {
	srv := newTestServer(t, []string{"http://127.0.0.1:1"})
	defer func() { _ = srv.Shutdown() }()

	req := httptest.NewRequest(http.MethodPut, "/strategy", bytes.NewBufferString(`{"strategy":"cost_aware"}`))
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/strategy", bytes.NewBufferString(`{"strategy":"session_affinity"}`))
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestStrategyEndpointRejectsUnsupportedMethods(t *testing.T) {
	srv := newTestServer(t, []string{"http://127.0.0.1:1"})
	defer func() { _ = srv.Shutdown() }()

	req := httptest.NewRequest(http.MethodPost, "/strategy", bytes.NewBufferString(`{"strategy":"round_robin"}`))
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
}

func TestMetricsEndpointIncludesStrategyAndBackendCounters(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "/infer":
			writeJSON(w, http.StatusOK, map[string]string{
				"model":       "mock-llm",
				"output_text": "Mock response to: hello",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()

	srv := newTestServer(t, []string{backend.URL})
	defer func() { _ = srv.Shutdown() }()

	body := proxy.ChatCompletionRequest{
		Model: "mock-llm",
		Messages: []proxy.ChatMessage{{
			Role:    "user",
			Content: "hello",
		}},
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", mustJSON(body))
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected chat completion status 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`inferflow_strategy_selections_total{strategy="round_robin"} 1`)) {
		t.Fatalf("expected strategy metric in body: %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`inferflow_backend_selections_total{backend="a"} 1`)) {
		t.Fatalf("expected backend metric in body: %s", rec.Body.String())
	}
}

func newTestServer(t *testing.T, backendURLs []string) *Server {
	t.Helper()
	backends := make([]*router.Backend, 0, len(backendURLs))
	for idx, raw := range backendURLs {
		backend, err := router.NewBackend(string(rune('a'+idx)), raw)
		if err != nil {
			t.Fatalf("new backend: %v", err)
		}
		backends = append(backends, backend)
	}

	srv, err := New(Config{
		ListenAddr:           ":0",
		Backends:             backends,
		ProbeInterval:        10 * time.Hour,
		BackendRequestTimout: time.Second,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	srv.probeBackends()
	return srv
}

func mustJSON(v any) *bytes.Buffer {
	data, _ := json.Marshal(v)
	return bytes.NewBuffer(data)
}
