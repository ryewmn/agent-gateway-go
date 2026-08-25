package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Address       string           `json:"address"`
	Timeout       Duration         `json:"timeout"`
	Retries       int              `json:"retries"`
	MaxConcurrent int              `json:"max_concurrent"`
	Providers     []ProviderConfig `json:"providers"`
}

type ProviderConfig struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Endpoint        string   `json:"endpoint,omitempty"`
	APIKeyEnv       string   `json:"api_key_env,omitempty"`
	Weight          float64  `json:"weight"`
	Latency         Duration `json:"latency,omitempty"`
	FailEvery       uint64   `json:"fail_every,omitempty"`
	FailFirst       uint64   `json:"fail_first,omitempty"`
	ToolQuality     int      `json:"tool_quality,omitempty"`
	FailureThreshold int     `json:"failure_threshold,omitempty"`
	CircuitCooldown Duration `json:"circuit_cooldown,omitempty"`
}

type Duration struct { time.Duration }
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b,&s); err != nil { return fmt.Errorf("duration must be a string: %w",err) }
	v,err := time.ParseDuration(s); if err != nil { return err }; d.Duration=v; return nil
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil { return Config{}, fmt.Errorf("read config: %w", err) }
	var c Config
	dec := json.NewDecoder(strings.NewReader(os.ExpandEnv(string(b))))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil { return c, fmt.Errorf("decode config: %w", err) }
	if err := c.Validate(); err != nil { return c, err }
	return c,nil
}

func (c Config) Validate() error {
	if c.Address == "" { return fmt.Errorf("address is required") }
	if len(c.Providers)==0 { return fmt.Errorf("at least one provider is required") }
	seen:=map[string]bool{}
	for i,p := range c.Providers {
		if p.Name=="" || seen[p.Name] { return fmt.Errorf("provider %d has empty or duplicate name",i) }; seen[p.Name]=true
		if p.Type!="mock" && p.Type!="openai-compatible" { return fmt.Errorf("provider %q has unsupported type %q",p.Name,p.Type) }
		if p.Type=="openai-compatible" && p.Endpoint=="" { return fmt.Errorf("provider %q requires endpoint",p.Name) }
		if p.Weight < 0 || p.ToolQuality < 0 || p.ToolQuality > 100 { return fmt.Errorf("provider %q has invalid weighting or quality",p.Name) }
	}
	return nil
}

func APIKey(p ProviderConfig) string { if p.APIKeyEnv=="" { return "" }; return os.Getenv(p.APIKeyEnv) }
