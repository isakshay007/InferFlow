package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

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
	strategyMu   sync.RWMutex
	strategy     router.Strategy
	strategyName string

	stopCh    chan struct{}
	stopOnce  sync.Once
	probeDone chan struct{}
}

func New(cfg Config) (*Server, error) {
	rr := router.NewRoundRobin(cfg.Backends)
	s := &Server{
		cfg:          cfg,
		client:       proxy.NewClient(cfg.BackendRequestTimout),
		metrics:      &metrics.State{},
		strategy:     rr,
		strategyName: rr.Name(),
		stopCh:       make(chan struct{}),
		probeDone:    make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/strategy", s.handleStrategy)
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)

	s.httpSrv = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
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

	estimatedCost := estimateRequestCost(req.Messages)
	ctx, strategySpan := otel.StartSpan(ctx, "strategy_evaluate")
	strategyName, strategy := s.activeStrategy()
	decision, err := strategy.Select(estimatedCost)
	if err == nil {
		strategySpan.SetAttribute("strategy_name", strategyName)
		strategySpan.SetAttribute("selected_backend", decision.Backend.Name)
		if strategyName == router.StrategyLeastPending {
			strategySpan.SetAttribute("pending_requests", decision.PendingRequests)
		}
		if strategyName == router.StrategyCostAware {
			strategySpan.SetAttribute("pending_cost", decision.PendingCost)
		}
	}
	strategySpan.End()
	if err != nil {
		http.Error(w, "no healthy backend available", http.StatusServiceUnavailable)
		return
	}

	backend := decision.Backend
	defer decision.Release()

	s.metrics.IncInFlight()
	defer s.metrics.DecInFlight()

	ctx, inferenceSpan := otel.StartSpan(ctx, "backend_inference")
	resp, err := s.client.SendChatCompletion(ctx, backend, req)
	inferenceSpan.End()
	if err != nil {
		backend.SetHealthy(false)
		http.Error(w, "backend request failed", http.StatusBadGateway)
		return
	}

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
	case router.StrategyCostAware:
		next = router.NewCostAware(s.cfg.Backends)
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
	case router.StrategyRoundRobin, router.StrategyLeastPending, router.StrategyCostAware:
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
