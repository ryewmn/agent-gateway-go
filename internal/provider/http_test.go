package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ryewmn/agent-gateway-go/internal/model"
)

func TestHTTPProviderResponseAndAuth(t *testing.T){
	ts:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.Header.Get("Authorization")!="Bearer secret"{t.Error("missing auth")};w.Header().Set("Content-Type","application/json");_,_=w.Write([]byte(`{"id":"x","model":"m","choices":[]}`))}));defer ts.Close()
	r,err:=NewHTTP("p",ts.URL,"secret",ts.Client()).Chat(context.Background(),model.ChatRequest{Model:"m"});if err!=nil{t.Fatal(err)};if r.ID!="x"{t.Fatalf("id=%s",r.ID)}
}
func TestHTTPProviderMapsRetryableStatus(t *testing.T){ts:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(http.StatusTooManyRequests)}));defer ts.Close();_,err:=NewHTTP("p",ts.URL,"",ts.Client()).Chat(context.Background(),model.ChatRequest{});if !errors.Is(err,ErrRateLimited){t.Fatalf("error=%v",err)}}
