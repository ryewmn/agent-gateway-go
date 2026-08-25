package observability

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMetricsConcurrentAndEscaped(t *testing.T){m:=NewMetrics();var wg sync.WaitGroup;for i:=0;i<20;i++{wg.Add(1);go func(){defer wg.Done();m.Observe(`unsafe"label`,"success",time.Millisecond)}()};wg.Wait();out:=m.Prometheus();if !strings.Contains(out,"unsafe_label")||!strings.Contains(out," 20"){t.Fatalf("metrics=%s",out)}}
