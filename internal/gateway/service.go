package gateway

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/model"
	"github.com/ryewmn/agent-gateway-go/internal/observability"
	"github.com/ryewmn/agent-gateway-go/internal/provider"
)

var ErrBusy = errors.New("gateway concurrency limit reached")

type Service struct {
	router *Router
	timeout time.Duration
	retries int
	sem chan struct{}
	metrics *observability.Metrics
}

func NewService(router *Router, timeout time.Duration, retries, maxConcurrent int, metrics *observability.Metrics) *Service {
	if timeout <= 0 { timeout = 5*time.Second }
	if retries < 0 { retries = 0 }
	if maxConcurrent < 1 { maxConcurrent = 32 }
	if metrics == nil { metrics = observability.NewMetrics() }
	return &Service{router: router, timeout: timeout, retries: retries, sem: make(chan struct{}, maxConcurrent), metrics: metrics}
}

func (s *Service) Chat(ctx context.Context, req model.ChatRequest) (model.ChatResponse, string, error) {
	select { case s.sem <- struct{}{}: defer func(){ <-s.sem }(); default: s.metrics.Busy(); return model.ChatResponse{}, "", ErrBusy }
	excluded := map[string]bool{}
	var last error
	for attempt := 0; attempt <= s.retries; attempt++ {
		route, err := s.router.Select(excluded, time.Now())
		if err != nil { if last != nil { return model.ChatResponse{}, "", last }; return model.ChatResponse{}, "", err }
		name := route.Provider.Name()
		started := time.Now()
		attemptCtx, cancel := context.WithTimeout(ctx, s.timeout)
		out, err := route.Provider.Chat(attemptCtx, req)
		cancel()
		elapsed := time.Since(started)
		route.observed(elapsed)
		if err == nil { route.Breaker.Success(); s.metrics.Observe(name, "success", elapsed); return out, name, nil }
		if ctx.Err() != nil {
			route.Breaker.CancelTrial()
			s.metrics.Observe(name, "canceled", elapsed)
			return model.ChatResponse{}, "", ctx.Err()
		}
		last = fmt.Errorf("provider %s failed: %w", name, err)
		if !provider.IsRetryable(err) {
			route.Breaker.CancelTrial()
			s.metrics.Observe(name, "rejected", elapsed)
			return model.ChatResponse{}, "", last
		}
		route.Breaker.Failure(time.Now()); s.metrics.Observe(name, "error", elapsed)
		excluded[name] = true
	}
	return model.ChatResponse{}, "", last
}

func (s *Service) Ready() bool { return s.router.Ready(time.Now()) }
func (s *Service) States() map[string]string { return s.router.States(time.Now()) }
func (s *Service) Metrics() *observability.Metrics { return s.metrics }
