package observability

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// Logger serializes JSON log records and deliberately accepts metadata only,
// not prompts, responses, authorization headers, or API keys.
type Logger struct { mu sync.Mutex; out io.Writer }
func NewLogger(out io.Writer) *Logger { return &Logger{out: out} }
func (l *Logger) Event(level, event string, fields map[string]any) {
	record := map[string]any{"time": time.Now().UTC().Format(time.RFC3339Nano), "level": level, "event": event}
	for k,v := range fields { record[k]=v }
	l.mu.Lock(); defer l.mu.Unlock(); _ = json.NewEncoder(l.out).Encode(record)
}
