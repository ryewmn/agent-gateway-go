package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	bench "github.com/ryewmn/agent-gateway-go/internal/benchmark"
	"github.com/ryewmn/agent-gateway-go/internal/gateway"
	"github.com/ryewmn/agent-gateway-go/internal/observability"
	"github.com/ryewmn/agent-gateway-go/internal/provider"
)

func main(){
	suitePath:=flag.String("suite","configs/benchmark.json","benchmark suite");quality:=flag.Int("quality",100,"deterministic mock tool quality 1-100");failEvery:=flag.Uint64("fail-every",0,"inject every Nth upstream failure");latency:=flag.Duration("latency",5*time.Millisecond,"mock latency")
	minExact:=flag.Float64("min-exact",0.95,"minimum exact tool-call accuracy");minAbstain:=flag.Float64("min-abstention",0.95,"minimum abstention accuracy");minMulti:=flag.Float64("min-multistep",0.90,"minimum multi-step success");maxP95:=flag.Duration("max-p95",250*time.Millisecond,"maximum p95 case latency");flag.Parse()
	suite,err:=bench.Load(*suitePath);if err!=nil{fatal(err)}
	primary:=provider.NewMock(provider.MockConfig{Name:"primary",Latency:*latency,FailEvery:*failEvery,ToolQuality:*quality})
	backup:=provider.NewMock(provider.MockConfig{Name:"backup",Latency:*latency+time.Millisecond,ToolQuality:*quality})
	router:=gateway.NewRouter(gateway.NewRoute(primary,1,nil),gateway.NewRoute(backup,0.8,nil))
	svc:=gateway.NewService(router,2*time.Second,1,8,observability.NewMetrics())
	report:=bench.Run(context.Background(),svc,suite,bench.Thresholds{MinExact:*minExact,MinAbstention:*minAbstain,MinMultiStep:*minMulti,MaxP95:*maxP95})
	b,_:=json.MarshalIndent(report,"","  ");fmt.Println(string(b));if !report.RegressionPassed{os.Exit(1)}
}
func fatal(err error){fmt.Fprintln(os.Stderr,err);os.Exit(2)}
