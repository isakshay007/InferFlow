package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"inferflow/internal/proxy"
)

type Generator interface {
	HealthCheck(context.Context) error
	Generate(context.Context, string) (string, error)
}

func NewHandler(client Generator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := client.HealthCheck(r.Context()); err != nil {
			http.Error(w, "triton unavailable", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/infer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req proxy.BackendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Model) == "" {
			http.Error(w, "model is required", http.StatusBadRequest)
			return
		}
		if len(req.Messages) == 0 {
			http.Error(w, "messages must not be empty", http.StatusBadRequest)
			return
		}

		prompt := flattenMessages(req.Messages)
		output, err := client.Generate(r.Context(), prompt)
		if err != nil {
			http.Error(w, "triton inference failed", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, proxy.BackendResponse{
			Model:      req.Model,
			OutputText: output,
		})
	})

	return mux
}

func flattenMessages(messages []proxy.ChatMessage) string {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		lines = append(lines, msg.Role+": "+content)
	}
	return strings.Join(lines, "\n")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
