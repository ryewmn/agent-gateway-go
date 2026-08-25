package gateway

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/model"
	"github.com/ryewmn/agent-gateway-go/internal/observability"
	"github.com/ryewmn/agent-gateway-go/internal/provider"
)

type stubProvider struct{name string; calls atomic.Int32; fail bool; gate <-chan struct{}}
func(s *stubProvider)Name()string{return s.name}
func(s *stubProvider)Chat(ctx context.Context,r model.ChatRequest)(model.ChatResponse,error){s.calls.Add(1);if s.gate!=nil{select{case<-s.gate:case<-ctx.Done():return model.ChatResponse{},ctx.Err()}};if s.fail{return model.ChatResponse{},provider.ErrUnavailable};return model.ChatResponse{Model:r.Model},nil}

func TestServiceRetriesDifferentProvider(t *testing.T){
	a:=&stubProvider{name:"a",fail:true};b:=&stubProvider{name:"b"}
	s:=NewService(NewRouter(NewRoute(a,2,nil),NewRoute(b,1,nil)),time.Second,1,2,observability.NewMetrics())
	_,name,err:=s.Chat(context.Background(),model.ChatRequest{Model:"m"});if err!=nil{t.Fatal(err)};if name!="b"{t.Fatalf("provider=%s",name)};if a.calls.Load()!=1||b.calls.Load()!=1{t.Fatal("expected one attempt per provider")}
}

func TestServiceConcurrencyLimit(t *testing.T){
	gate:=make(chan struct{});p:=&stubProvider{name:"blocked",gate:gate};s:=NewService(NewRouter(NewRoute(p,1,nil)),time.Second,0,1,nil)
	done:=make(chan struct{});go func(){_,_,_=s.Chat(context.Background(),model.ChatRequest{Model:"m"});close(done)}()
	deadline:=time.After(time.Second);for p.calls.Load()==0{select{case<-deadline:t.Fatal("first call did not start");default:time.Sleep(time.Millisecond)}}
	_,_,err:=s.Chat(context.Background(),model.ChatRequest{Model:"m"});if !errors.Is(err,ErrBusy){t.Fatalf("error=%v",err)};close(gate);<-done
}
