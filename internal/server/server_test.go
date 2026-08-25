package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/gateway"
	"github.com/ryewmn/agent-gateway-go/internal/observability"
	"github.com/ryewmn/agent-gateway-go/internal/provider"
)

func testServer(logs *bytes.Buffer)*Server{p:=provider.NewMock(provider.MockConfig{Name:"mock",ToolQuality:100});svc:=gateway.NewService(gateway.NewRouter(gateway.NewRoute(p,1,nil)),time.Second,0,2,nil);return New(svc,observability.NewLogger(logs))}
func TestChatCompletionAndSafeLog(t *testing.T){logs:=new(bytes.Buffer);s:=testServer(logs);secret:="private prompt text";body:=`{"model":"m","messages":[{"role":"user","content":"`+secret+`"}]}`;req:=httptest.NewRequest(http.MethodPost,"/v1/chat/completions",strings.NewReader(body));w:=httptest.NewRecorder();s.Handler().ServeHTTP(w,req);if w.Code!=http.StatusOK{t.Fatalf("status=%d body=%s",w.Code,w.Body.String())};if w.Header().Get("X-Request-ID")==""{t.Fatal("missing request id")};if strings.Contains(logs.String(),secret){t.Fatal("prompt leaked to logs")}}
func TestRejectsMultipleJSONValues(t *testing.T){s:=testServer(new(bytes.Buffer));req:=httptest.NewRequest(http.MethodPost,"/v1/chat/completions",strings.NewReader(`{"model":"m","messages":[{"role":"user","content":"x"}]} {}`));w:=httptest.NewRecorder();s.Handler().ServeHTTP(w,req);if w.Code!=http.StatusBadRequest{t.Fatalf("status=%d",w.Code)}}
func TestHealthAndMetrics(t *testing.T){s:=testServer(new(bytes.Buffer));for _,path:=range []string{"/healthz","/readyz","/metrics"}{w:=httptest.NewRecorder();s.Handler().ServeHTTP(w,httptest.NewRequest(http.MethodGet,path,nil));if w.Code!=http.StatusOK{t.Fatalf("%s status=%d",path,w.Code)}}}
