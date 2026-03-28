package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"inferflow/internal/adapter"
	"inferflow/internal/triton"
)

func main() {
	addr := getenv("TRITON_ADAPTER_ADDR", ":9000")
	tritonURL := getenv("TRITON_BASE_URL", "http://localhost:8000")
	modelName := getenv("TRITON_MODEL_NAME", "qwen3_0_6b")
	timeout := durationFromEnv("TRITON_TIMEOUT", 60*time.Second)
	maxNewTokens := intFromEnv("TRITON_MAX_NEW_TOKENS", 128)

	client := triton.NewClient(tritonURL, modelName, timeout, maxNewTokens)
	server := &http.Server{
		Addr:    addr,
		Handler: adapter.NewHandler(client),
	}

	log.Printf("triton adapter listening on %s, targeting %s model %s", addr, tritonURL, modelName)
	log.Fatal(server.ListenAndServe())
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intFromEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
