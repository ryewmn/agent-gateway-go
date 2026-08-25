package gateway

import (
	"errors"
	"sync"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/provider"
)

var ErrNoProvider = errors.New("no healthy provider available")

type Route struct {
	Provider provider.Provider
	Weight float64
	Breaker *CircuitBreaker
	mu sync.Mutex
	ewma time.Duration
	inflight int
}

func NewRoute(p provider.Provider, weight float64, breaker *CircuitBreaker) *Route {
	if weight <= 0 { weight = 1 }
	if breaker == nil { breaker = NewCircuitBreaker(3, 10*time.Second) }
	return &Route{Provider: p, Weight: weight, Breaker: breaker, ewma: 100*time.Millisecond}
}

func (r *Route) score() float64 {
	r.mu.Lock(); defer r.mu.Unlock()
	// Queue pressure prevents all concurrent requests choosing the same route.
	return float64(r.ewma) * float64(r.inflight+1) / r.Weight
}
func (r *Route) started() { r.mu.Lock(); r.inflight++; r.mu.Unlock() }
func (r *Route) observed(d time.Duration) {
	r.mu.Lock(); defer r.mu.Unlock()
	if r.inflight > 0 { r.inflight-- }
	r.ewma = time.Duration(float64(r.ewma)*0.8 + float64(d)*0.2)
}

type Router struct { routes []*Route }

func NewRouter(routes ...*Route) *Router { return &Router{routes: routes} }

func (r *Router) Select(exclude map[string]bool, now time.Time) (*Route, error) {
	// State inspection has no reservation side effect. Only the winning route
	// calls Allow, so unselected half-open circuits cannot be left stuck.
	rejected := map[string]bool{}
	for {
		var best *Route
		bestScore := 0.0
		for _, route := range r.routes {
			name := route.Provider.Name()
			if exclude[name] || rejected[name] || route.Breaker.State(now) == "open" { continue }
			s := route.score()
			if best == nil || s < bestScore { best, bestScore = route, s }
		}
		if best == nil { return nil, ErrNoProvider }
		if best.Breaker.Allow(now) {
			best.started()
			return best, nil
		}
		rejected[best.Provider.Name()] = true
	}
}

func (r *Router) Ready(now time.Time) bool {
	for _, route := range r.routes { if route.Breaker.State(now) != "open" { return true } }
	return false
}

func (r *Router) States(now time.Time) map[string]string {
	out := make(map[string]string, len(r.routes))
	for _, route := range r.routes { out[route.Provider.Name()] = route.Breaker.State(now) }
	return out
}
