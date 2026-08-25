package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/model"
)

func TestMockToolCallAndFailureInjection(t *testing.T){
	p:=NewMock(MockConfig{Name:"test",FailEvery:2,ToolQuality:100})
	req:=model.ChatRequest{Model:"m",Messages:[]model.Message{{Role:"user",Content:`CALL:lookup {"id":"42"}`}},Tools:[]model.Tool{{Type:"function",Function:model.FunctionSpec{Name:"lookup"}}}}
	r,err:=p.Chat(context.Background(),req);if err!=nil{t.Fatal(err)};if got:=r.Choices[0].Message.ToolCalls[0].Function.Name;got!="lookup"{t.Fatalf("name=%s",got)}
	_,err=p.Chat(context.Background(),req);if !errors.Is(err,ErrUnavailable){t.Fatalf("error=%v",err)}
}
func TestMockHonorsCancellation(t *testing.T){ctx,cancel:=context.WithTimeout(context.Background(),time.Millisecond);defer cancel();_,err:=NewMock(MockConfig{Latency:time.Second}).Chat(ctx,model.ChatRequest{});if !errors.Is(err,context.DeadlineExceeded){t.Fatalf("error=%v",err)}}
