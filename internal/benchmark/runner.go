package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/gateway"
	"github.com/ryewmn/agent-gateway-go/internal/model"
)

type Suite struct { Name string `json:"name"`; Cases []Case `json:"cases"` }
type Case struct { ID string `json:"id"`; Steps []Step `json:"steps"`; Tags []string `json:"tags,omitempty"` }
type Step struct {
	Prompt string `json:"prompt"`
	Tools []model.Tool `json:"tools,omitempty"`
	Expected *ExpectedCall `json:"expected_call,omitempty"`
	Abstain bool `json:"abstain,omitempty"`
}
type ExpectedCall struct { Name string `json:"name"`; Arguments map[string]any `json:"arguments"` }
type Thresholds struct { MinExact, MinAbstention, MinMultiStep float64; MaxP95 time.Duration }
type CaseResult struct { ID string `json:"id"`; Passed bool `json:"passed"`; Steps int `json:"steps"`; Error string `json:"error,omitempty"`; LatencyMS float64 `json:"latency_ms"` }
type Report struct {
	Suite string `json:"suite"`; Cases int `json:"cases"`; Passed int `json:"passed"`; Errors int `json:"errors"`
	ExactAccuracy float64 `json:"exact_accuracy"`; AbstentionAccuracy float64 `json:"abstention_accuracy"`; MultiStepSuccess float64 `json:"multi_step_success"`
	P50MS float64 `json:"p50_ms"`; P95MS float64 `json:"p95_ms"`; RegressionPassed bool `json:"regression_passed"`; Results []CaseResult `json:"results"`
}

func Load(path string) (Suite,error) {
	b,e:=os.ReadFile(path);if e!=nil{return Suite{},e};var s Suite;e=json.Unmarshal(b,&s);if e!=nil{return s,e}
	if s.Name==""||len(s.Cases)==0{return s,fmt.Errorf("suite name and cases are required")}
	for _,tc:=range s.Cases { if tc.ID==""||len(tc.Steps)==0{return s,fmt.Errorf("every case requires an id and at least one step")};for _,step:=range tc.Steps{if (step.Expected==nil)==!step.Abstain{return s,fmt.Errorf("case %q steps must set exactly one expected outcome",tc.ID)}} }
	return s,nil
}

func Run(ctx context.Context, svc *gateway.Service, suite Suite, thresholds Thresholds) Report {
	r:=Report{Suite:suite.Name,Cases:len(suite.Cases)}
	var exactOK,exactN,abstainOK,abstainN,multiOK,multiN int
	latencies:=make([]time.Duration,0,len(suite.Cases))
	for _,tc:=range suite.Cases {
		start:=time.Now(); cr:=CaseResult{ID:tc.ID,Steps:len(tc.Steps),Passed:true}
		if len(tc.Steps)>1 { multiN++ }
		for _,step:=range tc.Steps {
			resp,_,err:=svc.Chat(ctx,model.ChatRequest{Model:"benchmark-model",Messages:[]model.Message{{Role:"user",Content:step.Prompt}},Tools:step.Tools})
			if err!=nil { cr.Passed=false; cr.Error=err.Error(); r.Errors++; break }
			var calls []model.ToolCall; if len(resp.Choices)>0 { calls=resp.Choices[0].Message.ToolCalls }
			if step.Abstain { abstainN++; ok:=len(calls)==0; if ok { abstainOK++ } else { cr.Passed=false }; continue }
			if step.Expected!=nil { exactN++; ok:=exactMatch(calls,*step.Expected); if ok { exactOK++ } else { cr.Passed=false } }
		}
		d:=time.Since(start);cr.LatencyMS=float64(d.Microseconds())/1000;latencies=append(latencies,d)
		if cr.Passed { r.Passed++; if len(tc.Steps)>1 {multiOK++} }
		r.Results=append(r.Results,cr)
	}
	r.ExactAccuracy=ratio(exactOK,exactN);r.AbstentionAccuracy=ratio(abstainOK,abstainN);r.MultiStepSuccess=ratio(multiOK,multiN)
	r.P50MS=float64(percentile(latencies,0.50).Microseconds())/1000;r.P95MS=float64(percentile(latencies,0.95).Microseconds())/1000
	r.RegressionPassed=r.Errors==0&&r.ExactAccuracy>=thresholds.MinExact&&r.AbstentionAccuracy>=thresholds.MinAbstention&&r.MultiStepSuccess>=thresholds.MinMultiStep&&(thresholds.MaxP95<=0||time.Duration(r.P95MS*float64(time.Millisecond))<=thresholds.MaxP95)
	return r
}

func exactMatch(calls []model.ToolCall,want ExpectedCall) bool {
	if len(calls)!=1||calls[0].Function.Name!=want.Name{return false}
	var got map[string]any;if json.Unmarshal([]byte(calls[0].Function.Arguments),&got)!=nil{return false};return reflect.DeepEqual(got,want.Arguments)
}
func ratio(a,b int) float64 {if b==0{return 1};return float64(a)/float64(b)}
func percentile(in []time.Duration,p float64)time.Duration{if len(in)==0{return 0};v:=append([]time.Duration(nil),in...);sort.Slice(v,func(i,j int)bool{return v[i]<v[j]});i:=int(float64(len(v)-1)*p);return v[i]}
