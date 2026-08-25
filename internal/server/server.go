package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/gateway"
	"github.com/ryewmn/agent-gateway-go/internal/model"
	"github.com/ryewmn/agent-gateway-go/internal/observability"
)

type Server struct { service *gateway.Service; logger *observability.Logger; maxBody int64; mux *http.ServeMux }

func New(service *gateway.Service, logger *observability.Logger) *Server {
	s := &Server{service:service, logger:logger, maxBody:1<<20, mux:http.NewServeMux()}
	s.mux.HandleFunc("POST /v1/chat/completions",s.chat)
	s.mux.HandleFunc("GET /healthz",s.health)
	s.mux.HandleFunc("GET /readyz",s.ready)
	s.mux.HandleFunc("GET /metrics",s.metrics)
	return s
}
func (s *Server) Handler() http.Handler { return s.securityHeaders(s.mux) }

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	id := requestID(); started := time.Now(); w.Header().Set("X-Request-ID",id)
	r.Body = http.MaxBytesReader(w,r.Body,s.maxBody)
	dec:=json.NewDecoder(r.Body)
	var req model.ChatRequest
	if err:=dec.Decode(&req); err!=nil { s.fail(w,id,http.StatusBadRequest,"invalid_request",friendlyDecode(err)); return }
	if err:=ensureEOF(dec); err!=nil { s.fail(w,id,http.StatusBadRequest,"invalid_request","request body must contain one JSON object"); return }
	if err:=validate(req); err!=nil { s.fail(w,id,http.StatusBadRequest,"invalid_request",err.Error()); return }
	out,providerName,err:=s.service.Chat(r.Context(),req)
	if err!=nil {
		status,code:=http.StatusBadGateway,"upstream_error"
		if errors.Is(err,gateway.ErrBusy) { status,code=http.StatusServiceUnavailable,"busy" }
		if errors.Is(err,gateway.ErrNoProvider) { status,code=http.StatusServiceUnavailable,"no_provider" }
		s.logger.Event("error","chat_failed",map[string]any{"request_id":id,"code":code,"duration_ms":time.Since(started).Milliseconds()})
		s.fail(w,id,status,code,"request could not be completed"); return
	}
	s.logger.Event("info","chat_completed",map[string]any{"request_id":id,"provider":providerName,"duration_ms":time.Since(started).Milliseconds()})
	writeJSON(w,http.StatusOK,out)
}

func validate(r model.ChatRequest) error {
	if strings.TrimSpace(r.Model)=="" { return fmt.Errorf("model is required") }
	if len(r.Messages)==0 || len(r.Messages)>100 { return fmt.Errorf("messages must contain 1 to 100 items") }
	for _,m:=range r.Messages { if m.Role!="system"&&m.Role!="user"&&m.Role!="assistant"&&m.Role!="tool" { return fmt.Errorf("unsupported message role") }; if len(m.Content)>100_000 { return fmt.Errorf("message is too large") } }
	if len(r.Tools)>128 { return fmt.Errorf("too many tools") }
	for _,tool:=range r.Tools { if tool.Type!="function" || strings.TrimSpace(tool.Function.Name)=="" { return fmt.Errorf("tools must contain named functions") } }
	if r.Temperature!=nil && (*r.Temperature<0 || *r.Temperature>2) { return fmt.Errorf("temperature must be between 0 and 2") }
	return nil
}
func ensureEOF(dec *json.Decoder) error { var extra any; err:=dec.Decode(&extra); if errors.Is(err,io.EOF){return nil}; if err==nil{return fmt.Errorf("multiple JSON values")}; return err }
func friendlyDecode(err error) string { var max *http.MaxBytesError; if errors.As(err,&max){return "request body exceeds 1 MiB"}; return "request body is not valid JSON" }
func (s *Server) health(w http.ResponseWriter,_ *http.Request){ writeJSON(w,http.StatusOK,map[string]string{"status":"ok"}) }
func (s *Server) ready(w http.ResponseWriter,_ *http.Request){ if !s.service.Ready(){writeJSON(w,http.StatusServiceUnavailable,map[string]any{"status":"not_ready","providers":s.service.States()});return}; writeJSON(w,http.StatusOK,map[string]any{"status":"ready","providers":s.service.States()}) }
func (s *Server) metrics(w http.ResponseWriter,_ *http.Request){ w.Header().Set("Content-Type","text/plain; version=0.0.4"); _,_=io.WriteString(w,s.service.Metrics().Prometheus()) }
func (s *Server) fail(w http.ResponseWriter,id string,status int,code,msg string){ writeJSON(w,status,model.ErrorBody{Error:model.APIError{Message:msg,Type:"gateway_error",Code:code,RequestID:id}}) }
func writeJSON(w http.ResponseWriter,status int,v any){ w.Header().Set("Content-Type","application/json"); w.WriteHeader(status); _=json.NewEncoder(w).Encode(v) }
func requestID() string { var b [12]byte; if _,err:=rand.Read(b[:]);err!=nil{return fmt.Sprintf("req_%d",time.Now().UnixNano())}; return "req_"+hex.EncodeToString(b[:]) }
func (s *Server) securityHeaders(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){ w.Header().Set("X-Content-Type-Options","nosniff"); w.Header().Set("Cache-Control","no-store"); next.ServeHTTP(w,r) }) }
