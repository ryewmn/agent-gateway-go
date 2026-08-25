package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/gateway"
	"github.com/ryewmn/agent-gateway-go/internal/model"
	"github.com/ryewmn/agent-gateway-go/internal/provider"
)

func TestRunObjectiveScores(t *testing.T){
	p:=provider.NewMock(provider.MockConfig{Name:"mock",ToolQuality:100});svc:=gateway.NewService(gateway.NewRouter(gateway.NewRoute(p,1,nil)),time.Second,0,1,nil)
	tool:=model.Tool{Type:"function",Function:model.FunctionSpec{Name:"lookup"}}
	s:=Suite{Name:"test",Cases:[]Case{{ID:"tool",Steps:[]Step{{Prompt:`CALL:lookup {"id":"1"}`,Tools:[]model.Tool{tool},Expected:&ExpectedCall{Name:"lookup",Arguments:map[string]any{"id":"1"}}}}},{ID:"abstain",Steps:[]Step{{Prompt:"answer directly",Tools:[]model.Tool{tool},Abstain:true}}}}}
	r:=Run(context.Background(),svc,s,Thresholds{MinExact:1,MinAbstention:1,MinMultiStep:1,MaxP95:time.Second});if !r.RegressionPassed||r.Passed!=2||r.ExactAccuracy!=1||r.AbstentionAccuracy!=1{t.Fatalf("report=%+v",r)}
}
