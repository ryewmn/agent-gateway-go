package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/config"
	"github.com/ryewmn/agent-gateway-go/internal/gateway"
	"github.com/ryewmn/agent-gateway-go/internal/observability"
	"github.com/ryewmn/agent-gateway-go/internal/provider"
	"github.com/ryewmn/agent-gateway-go/internal/server"
)

func main(){
	configPath:=flag.String("config","configs/local.json","path to JSON configuration");flag.Parse()
	cfg,err:=config.Load(*configPath);if err!=nil{log.Fatal(err)}
	routes:=make([]*gateway.Route,0,len(cfg.Providers))
	for _,pc:=range cfg.Providers {
		var p provider.Provider
		switch pc.Type {case "mock":p=provider.NewMock(provider.MockConfig{Name:pc.Name,Latency:pc.Latency.Duration,FailEvery:pc.FailEvery,FailFirst:pc.FailFirst,ToolQuality:pc.ToolQuality});case "openai-compatible":p=provider.NewHTTP(pc.Name,pc.Endpoint,config.APIKey(pc),&http.Client{})}
		routes=append(routes,gateway.NewRoute(p,pc.Weight,gateway.NewCircuitBreaker(pc.FailureThreshold,pc.CircuitCooldown.Duration)))
	}
	metrics:=observability.NewMetrics();svc:=gateway.NewService(gateway.NewRouter(routes...),cfg.Timeout.Duration,cfg.Retries,cfg.MaxConcurrent,metrics)
	app:=server.New(svc,observability.NewLogger(os.Stdout))
	httpServer:=&http.Server{Addr:cfg.Address,Handler:app.Handler(),ReadHeaderTimeout:5*time.Second,IdleTimeout:60*time.Second}
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
	go func(){<-ctx.Done();shutdownCtx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel();_=httpServer.Shutdown(shutdownCtx)}()
	log.Printf("gateway listening on %s",cfg.Address)
	if err:=httpServer.ListenAndServe();err!=nil&&!errors.Is(err,http.ErrServerClosed){log.Fatal(err)}
}
