package gateway

import (
	"testing"
	"time"
)

func TestCircuitBreakerLifecycle(t *testing.T) {
	now:=time.Unix(100,0); c:=NewCircuitBreaker(2,time.Second)
	if !c.Allow(now){t.Fatal("closed breaker rejected request")}
	c.Failure(now);if !c.Allow(now){t.Fatal("breaker opened before threshold")};c.Failure(now)
	if c.Allow(now){t.Fatal("open breaker admitted request")}
	if !c.Allow(now.Add(2*time.Second)){t.Fatal("half-open breaker rejected trial")}
	if c.Allow(now.Add(2*time.Second)){t.Fatal("half-open breaker admitted concurrent trial")}
	c.Success();if got:=c.State(now);got!="closed"{t.Fatalf("state=%s",got)}
}
