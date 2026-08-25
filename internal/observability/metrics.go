package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type metric struct { count uint64; seconds float64 }
type Metrics struct {
	mu sync.Mutex
	requests map[string]metric
	busy atomic.Uint64
}

func NewMetrics() *Metrics { return &Metrics{requests: map[string]metric{}} }
func (m *Metrics) Busy() { m.busy.Add(1) }
func (m *Metrics) Observe(provider, status string, d time.Duration) {
	key := safe(provider)+"|"+safe(status)
	m.mu.Lock(); v := m.requests[key]; v.count++; v.seconds += d.Seconds(); m.requests[key] = v; m.mu.Unlock()
}

func (m *Metrics) Prometheus() string {
	m.mu.Lock(); snapshot := make(map[string]metric, len(m.requests)); for k,v := range m.requests { snapshot[k]=v }; m.mu.Unlock()
	keys := make([]string,0,len(snapshot)); for k := range snapshot { keys=append(keys,k) }; sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# HELP agent_gateway_requests_total Completed upstream attempts.\n# TYPE agent_gateway_requests_total counter\n")
	for _, k := range keys { p := strings.SplitN(k,"|",2); fmt.Fprintf(&b,"agent_gateway_requests_total{provider=%q,status=%q} %d\n",p[0],p[1],snapshot[k].count) }
	b.WriteString("# HELP agent_gateway_upstream_duration_seconds_sum Total upstream time.\n# TYPE agent_gateway_upstream_duration_seconds_sum counter\n")
	for _, k := range keys { p := strings.SplitN(k,"|",2); fmt.Fprintf(&b,"agent_gateway_upstream_duration_seconds_sum{provider=%q,status=%q} %.6f\n",p[0],p[1],snapshot[k].seconds) }
	fmt.Fprintf(&b,"# HELP agent_gateway_busy_total Requests rejected by concurrency limit.\n# TYPE agent_gateway_busy_total counter\nagent_gateway_busy_total %d\n",m.busy.Load())
	return b.String()
}

func safe(s string) string { return strings.Map(func(r rune) rune { if r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||r=='_'||r=='-' { return r }; return '_' }, s) }
