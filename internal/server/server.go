package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"inferflow/internal/cache"
	"inferflow/internal/metrics"
	"inferflow/internal/otel"
	"inferflow/internal/proxy"
	"inferflow/internal/router"
)

type Server struct {
	cfg          Config
	httpSrv      *http.Server
	client       *proxy.Client
	metrics      *metrics.State
	cacheStore   cache.Store
	cacheTTL     time.Duration
	strategyMu   sync.RWMutex
	strategy     router.Strategy
	strategyName string

	stopCh    chan struct{}
	stopOnce  sync.Once
	probeDone chan struct{}
}

func New(cfg Config) (*Server, error) {
	rr := router.NewRoundRobin(cfg.Backends)
	store := cfg.AffinityStore
	if store == nil {
		store = cache.NewMemoryStore()
	}
	s := &Server{
		cfg:          cfg,
		client:       proxy.NewClient(cfg.BackendRequestTimout),
		metrics:      &metrics.State{},
		cacheStore:   store,
		cacheTTL:     cfg.CacheTTL,
		strategy:     rr,
		strategyName: rr.Name(),
		stopCh:       make(chan struct{}),
		probeDone:    make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/strategy", s.handleStrategy)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)

	s.httpSrv = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: corsMiddleware(mux),
	}

	s.startProber()
	return s, nil
}

func (s *Server) Run() error {
	err := s.httpSrv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown() error {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		<-s.probeDone
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"in_flight": s.metrics.InFlight(),
	})
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	_, strategy := s.activeStrategy()
	if !strategy.HasHealthyBackend() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "no healthy backend"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleStrategy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name, _ := s.activeStrategy()
		writeJSON(w, http.StatusOK, map[string]string{"strategy": name})
	case http.MethodPut:
		var req struct {
			Strategy string `json:"strategy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		name, err := s.setStrategy(req.Strategy)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"strategy": name})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	fmt.Fprintf(w, "# HELP inferflow_inflight_requests Current in-flight requests.\n")
	fmt.Fprintf(w, "# TYPE inferflow_inflight_requests gauge\n")
	fmt.Fprintf(w, "inferflow_inflight_requests %d\n", s.metrics.InFlight())

	fmt.Fprintf(w, "# HELP inferflow_requests_total Total requests accepted by the router.\n")
	fmt.Fprintf(w, "# TYPE inferflow_requests_total counter\n")
	fmt.Fprintf(w, "inferflow_requests_total %d\n", s.metrics.RequestsTotal())

	fmt.Fprintf(w, "# HELP inferflow_backend_errors_total Total backend request errors.\n")
	fmt.Fprintf(w, "# TYPE inferflow_backend_errors_total counter\n")
	fmt.Fprintf(w, "inferflow_backend_errors_total %d\n", s.metrics.BackendErrors())

	strategies := s.metrics.StrategySnapshot()
	fmt.Fprintf(w, "# HELP inferflow_strategy_selections_total Total strategy selections.\n")
	fmt.Fprintf(w, "# TYPE inferflow_strategy_selections_total counter\n")
	for _, name := range s.metrics.SortedKeys(strategies) {
		fmt.Fprintf(w, "inferflow_strategy_selections_total{strategy=%q} %d\n", name, strategies[name])
	}

	backends := s.metrics.BackendSnapshot()
	fmt.Fprintf(w, "# HELP inferflow_backend_selections_total Total backend selections.\n")
	fmt.Fprintf(w, "# TYPE inferflow_backend_selections_total counter\n")
	for _, name := range s.metrics.SortedKeys(backends) {
		fmt.Fprintf(w, "inferflow_backend_selections_total{backend=%q} %d\n", name, backends[name])
	}
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, span := otel.StartSpan(r.Context(), "router_receive")
	defer span.End()

	var req proxy.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := validateChatRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.metrics.IncRequestsTotal()

	estimatedCost := estimateRequestCost(req.Messages)
	cacheKey := buildCacheKey(req.Model, req.Messages)
	ctx, strategySpan := otel.StartSpan(ctx, "strategy_evaluate")
	strategyName, strategy := s.activeStrategy()
	decision, err := strategy.Select(router.SelectionInput{
		Context:       ctx,
		EstimatedCost: estimatedCost,
		CacheKey:      cacheKey,
	})
	if err == nil {
		strategySpan.SetAttribute("strategy_name", strategyName)
		strategySpan.SetAttribute("selected_backend", decision.Backend.Name)
		if strategyName == router.StrategyLeastPending {
			strategySpan.SetAttribute("pending_requests", decision.PendingRequests)
		}
		if strategyName == router.StrategyKVAware {
			strategySpan.SetAttribute("cache_key_present", cacheKey != "")
		}
	}
	strategySpan.End()
	if err != nil {
		http.Error(w, "no healthy backend available", http.StatusServiceUnavailable)
		return
	}

	backend := decision.Backend
	defer decision.Release()
	s.metrics.RecordStrategy(strategyName)
	s.metrics.RecordBackend(backend.Name)
	if strategyName == router.StrategyKVAware {
		if decision.CacheHit {
			s.metrics.IncKVCacheHit()
		} else {
			s.metrics.IncKVCacheMiss()
		}
	}
	w.Header().Set("X-Inferflow-Backend", backend.Name)
	w.Header().Set("X-Inferflow-Strategy", strategyName)
	if strategyName == router.StrategyKVAware {
		if decision.CacheHit {
			w.Header().Set("X-Inferflow-Cache-Hit", "true")
		} else {
			w.Header().Set("X-Inferflow-Cache-Hit", "false")
		}
	}

	s.metrics.IncInFlight()
	defer s.metrics.DecInFlight()

	inferenceStart := time.Now()
	ctx, inferenceSpan := otel.StartSpan(ctx, "backend_inference")
	resp, err := s.client.SendChatCompletion(ctx, backend, req)
	inferenceSpan.End()
	s.metrics.RecordLatency(backend.Name, float64(time.Since(inferenceStart).Milliseconds()))
	if err != nil {
		backend.SetHealthy(false)
		s.metrics.IncBackendErrors()
		http.Error(w, "backend request failed", http.StatusBadGateway)
		return
	}

	s.recordAffinity(ctx, cacheKey, backend.Name)

	_, streamSpan := otel.StartSpan(ctx, "response_stream")
	defer streamSpan.End()
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) startProber() {
	go func() {
		defer close(s.probeDone)

		s.probeBackends()
		ticker := time.NewTicker(s.cfg.ProbeInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.probeBackends()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *Server) probeBackends() {
	for _, backend := range s.cfg.Backends {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := s.client.HealthCheck(ctx, backend)
		cancel()
		backend.SetHealthy(err == nil)
	}
}

func (s *Server) activeStrategy() (string, router.Strategy) {
	s.strategyMu.RLock()
	defer s.strategyMu.RUnlock()
	return s.strategyName, s.strategy
}

func (s *Server) setStrategy(raw string) (string, error) {
	name, err := normalizeStrategyName(raw)
	if err != nil {
		return "", err
	}

	var next router.Strategy
	switch name {
	case router.StrategyRoundRobin:
		next = router.NewRoundRobin(s.cfg.Backends)
	case router.StrategyLeastPending:
		next = router.NewLeastPending(s.cfg.Backends)
	case router.StrategyRandom:
		next = router.NewRandom(s.cfg.Backends)
	case router.StrategyKVAware:
		next = router.NewKVAware(s.cfg.Backends, s.cacheStore)
	default:
		return "", errors.New("unsupported strategy")
	}

	s.strategyMu.Lock()
	s.strategy = next
	s.strategyName = name
	s.strategyMu.Unlock()

	return name, nil
}

func normalizeStrategyName(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	name = strings.ReplaceAll(name, "-", "_")
	if name == "" {
		return "", errors.New("strategy is required")
	}

	switch name {
	case router.StrategyRoundRobin, router.StrategyLeastPending, router.StrategyRandom, router.StrategyKVAware:
		return name, nil
	default:
		return "", errors.New("unsupported strategy")
	}
}

func validateChatRequest(req proxy.ChatCompletionRequest) error {
	if strings.TrimSpace(req.Model) == "" {
		return errors.New("model is required")
	}
	if len(req.Messages) == 0 {
		return errors.New("messages must not be empty")
	}
	return nil
}

func estimateRequestCost(messages []proxy.ChatMessage) int {
	totalBytes := 0
	for _, msg := range messages {
		totalBytes += len(strings.TrimSpace(msg.Content))
	}
	if totalBytes == 0 {
		return 1
	}

	cost := totalBytes / 4
	if totalBytes%4 != 0 {
		cost++
	}
	if cost == 0 {
		return 1
	}
	return cost
}

func buildCacheKey(model string, messages []proxy.ChatMessage) string {
	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(model))
	builder.WriteByte('\n')
	for _, msg := range messages {
		builder.WriteString(strings.TrimSpace(msg.Role))
		builder.WriteByte(':')
		builder.WriteString(strings.TrimSpace(msg.Content))
		builder.WriteByte('\n')
	}
	payload := strings.TrimSpace(builder.String())
	if payload == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(payload))
	return "inferflow:prefix:" + hex.EncodeToString(sum[:16])
}

func (s *Server) recordAffinity(ctx context.Context, cacheKey, backendName string) {
	if s.cacheStore == nil || cacheKey == "" || backendName == "" {
		return
	}
	_ = s.cacheStore.RememberBackend(ctx, cacheKey, backendName, s.cacheTTL)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	strategyName, _ := s.activeStrategy()
	latency := s.metrics.LatencySnapshot()
	backendSelections := s.metrics.BackendSnapshot()

	type backendStatus struct {
		Name      string `json:"name"`
		Healthy   bool   `json:"healthy"`
		Pending   int64  `json:"pending"`
		LatencyMs int64  `json:"latency_ms"`
	}
	backends := make([]backendStatus, 0, len(s.cfg.Backends))
	for _, b := range s.cfg.Backends {
		backends = append(backends, backendStatus{
			Name:      b.Name,
			Healthy:   b.Healthy(),
			Pending:   backendSelections[b.Name],
			LatencyMs: latency[b.Name],
		})
	}

	hits := s.metrics.KVCacheHits()
	misses := s.metrics.KVCacheMisses()
	var hitRate float64
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"strategy": strategyName,
		"backends": backends,
		"metrics": map[string]any{
			"requests_total":    s.metrics.RequestsTotal(),
			"in_flight":         s.metrics.InFlight(),
			"backend_errors":    s.metrics.BackendErrors(),
			"kv_cache_hits":     hits,
			"kv_cache_misses":   misses,
			"kv_cache_hit_rate": hitRate,
		},
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
