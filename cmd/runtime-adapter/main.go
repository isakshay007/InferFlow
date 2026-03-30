package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"inferflow/internal/runtime/ollama"
)

type inferRequest struct {
	Model    string              `json:"model"`
	Messages []map[string]string `json:"messages"`
	Stream   bool                `json:"stream"`
}

type inferResponse struct {
	Model      string `json:"model"`
	OutputText string `json:"output_text"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	addr := getenv("ADAPTER_ADDR", ":9000")
	baseURL := getenv("LLM_BASE_URL", "http://localhost:11434")
	defaultModel := getenv("LLM_MODEL", "qwen2.5:0.5b")
	timeout := durationFromEnv("LLM_TIMEOUT", 60*time.Second)

	client := ollama.NewClient(baseURL, defaultModel, timeout)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := client.Health(r.Context()); err != nil {
			http.Error(w, "ollama unavailable", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/infer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req inferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if len(req.Messages) == 0 {
			http.Error(w, "messages must not be empty", http.StatusBadRequest)
			return
		}

		text, err := client.Generate(r.Context(), req.Messages)
		if err != nil {
			http.Error(w, "ollama inference failed", http.StatusBadGateway)
			return
		}

		model := strings.TrimSpace(req.Model)
		if model == "" {
			model = defaultModel
		}

		writeJSON(w, http.StatusOK, inferResponse{
			Model:      model,
			OutputText: text,
		})
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("runtime adapter listening on %s (ollama: %s, model: %s)", addr, baseURL, defaultModel)
	log.Fatal(server.ListenAndServe())
}

func runHealthcheck() int {
	addr := getenv("ADAPTER_ADDR", ":9000")
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+host+"/healthz", nil)
	if err != nil {
		return 1
	}
	resp, err := client.Do(req)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	if secs, err := strconv.Atoi(value); err == nil {
		return time.Duration(secs) * time.Second
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
