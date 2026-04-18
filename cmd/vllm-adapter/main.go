package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"inferflow/internal/adapter"
	"inferflow/internal/vllm"
)

func main() {
	addr := getenv("VLLM_ADAPTER_ADDR", ":9000")
	vllmURL := getenv("VLLM_BASE_URL", "http://localhost:8000")
	modelName := getenv("VLLM_MODEL_NAME", "Qwen/Qwen2.5-0.5B-Instruct")
	timeout := durationFromEnv("VLLM_TIMEOUT", 60*time.Second)
	maxTokens := intFromEnv("VLLM_MAX_TOKENS", 128)
	temperature := floatFromEnv("VLLM_TEMPERATURE", 0.0)

	client := vllm.NewClient(vllmURL, modelName, timeout, maxTokens, temperature)
	server := &http.Server{
		Addr:    addr,
		Handler: adapter.NewHandler(client),
	}

	log.Printf("vllm adapter listening on %s, targeting %s model %s", addr, vllmURL, modelName)
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

func floatFromEnv(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
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
