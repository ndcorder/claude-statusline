package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	configPath string
	userHomeDir = os.UserHomeDir
)

type Config struct {
	UserHost       bool             `json:"user_host"`
	Cwd            bool             `json:"cwd"`
	Model          bool             `json:"model"`
	Git            GitConfig        `json:"git"`
	ContextBar     ContextBarConfig `json:"context_bar"`
	Cost           CostConfig       `json:"cost"`
	Duration       DurationConfig   `json:"duration"`
	Tokens         TokensConfig     `json:"tokens"`
	Cache          CacheConfig      `json:"cache"`
	ContextRunway  bool             `json:"context_runway"`
	RateLimits     RateLimitConfig  `json:"rate_limits"`
	LineDeltas     bool             `json:"line_deltas"`
	ApiStats       ApiStatsConfig   `json:"api_stats"`
	SessionCompare bool             `json:"session_compare"`
	Pricing        PricingConfig    `json:"pricing"`
	Session        SessionConfig    `json:"session"`
}

type GitConfig struct {
	Enabled     bool `json:"enabled"`
	AheadBehind bool `json:"ahead_behind"`
	Changes     bool `json:"changes"`
	CacheTTL    int  `json:"cache_ttl"`
}

type PricingConfig struct {
	Opus   ModelPricing `json:"opus"`
	Sonnet ModelPricing `json:"sonnet"`
	Haiku  ModelPricing `json:"haiku"`
}

type ModelPricing struct {
	CacheReadRate   float64 `json:"cache_read_rate"`
	CacheCreateRate float64 `json:"cache_create_rate"`
}

type SessionConfig struct {
	Path        string `json:"path"`
	HistoryPath string `json:"history_path"`
	MaxHistory  int    `json:"max_history"`
}

type ContextBarConfig struct {
	Enabled bool `json:"enabled"`
	Width   int  `json:"width"`
}

type CostConfig struct {
	Enabled      bool   `json:"enabled"`
	Precision    int    `json:"precision"`
	Velocity     bool   `json:"velocity"`
	VelocityUnit string `json:"velocity_unit"`
}

type DurationConfig struct {
	Enabled bool   `json:"enabled"`
	Format  string `json:"format"`
}

type TokensConfig struct {
	Enabled bool   `json:"enabled"`
	Format  string `json:"format"`
}

type CacheConfig struct {
	Enabled        bool `json:"enabled"`
	Cumulative     bool `json:"cumulative"`
	Savings        bool `json:"savings"`
	Sparkline      bool `json:"sparkline"`
	SparklineWidth int  `json:"sparkline_width"`
}

type RateLimitConfig struct {
	Enabled   bool `json:"enabled"`
	ShowReset bool `json:"show_reset"`
	Show7Day  bool `json:"show_7day"`
}

type ApiStatsConfig struct {
	Enabled    bool `json:"enabled"`
	Throughput bool `json:"throughput"`
}

func defaultConfig() Config {
	return Config{
		UserHost: true,
		Cwd:      true,
		Model:    true,
		Git: GitConfig{
			Enabled:     true,
			AheadBehind: true,
			Changes:     true,
			CacheTTL:    5,
		},
		ContextBar: ContextBarConfig{
			Enabled: true,
			Width:   10,
		},
		Cost: CostConfig{
			Enabled:      true,
			Precision:    2,
			Velocity:     true,
			VelocityUnit: "hour",
		},
		Duration: DurationConfig{
			Enabled: true,
			Format:  "auto",
		},
		Tokens: TokensConfig{
			Enabled: true,
			Format:  "auto",
		},
		Cache: CacheConfig{
			Enabled:        true,
			Cumulative:     true,
			Savings:        true,
			Sparkline:      true,
			SparklineWidth: 8,
		},
		ContextRunway: true,
		RateLimits: RateLimitConfig{
			Enabled:   true,
			ShowReset: true,
			Show7Day:  true,
		},
		LineDeltas: true,
		ApiStats: ApiStatsConfig{
			Enabled:    true,
			Throughput: true,
		},
		SessionCompare: true,
		Pricing: PricingConfig{
			Opus:   ModelPricing{CacheReadRate: 13.50, CacheCreateRate: 3.75},
			Sonnet: ModelPricing{CacheReadRate: 2.70, CacheCreateRate: 0.75},
			Haiku:  ModelPricing{CacheReadRate: 0.72, CacheCreateRate: 0.20},
		},
		Session: SessionConfig{
			Path:        filepath.Join(os.TempDir(), "claude-statusline-session-go"),
			HistoryPath: filepath.Join(os.TempDir(), "claude-statusline-history"),
			MaxHistory:  50,
		},
	}
}

func (c *Config) validate(w io.Writer) {
	warn := func(msg string) {
		fmt.Fprintf(w, "claude-statusline: warning: %s\n", msg)
	}

	switch c.Cost.VelocityUnit {
	case "hour", "minute":
	default:
		warn(fmt.Sprintf("invalid velocity_unit %q, using \"hour\"", c.Cost.VelocityUnit))
		c.Cost.VelocityUnit = "hour"
	}

	switch c.Duration.Format {
	case "auto", "seconds", "minutes", "hours":
	default:
		warn(fmt.Sprintf("invalid duration.format %q, using \"auto\"", c.Duration.Format))
		c.Duration.Format = "auto"
	}

	switch c.Tokens.Format {
	case "auto", "raw", "k", "M":
	default:
		warn(fmt.Sprintf("invalid tokens.format %q, using \"auto\"", c.Tokens.Format))
		c.Tokens.Format = "auto"
	}

	if c.ContextBar.Width <= 0 {
		warn(fmt.Sprintf("invalid context_bar.width %d, using 10", c.ContextBar.Width))
		c.ContextBar.Width = 10
	}

	if c.Cost.Precision < 0 {
		warn(fmt.Sprintf("invalid cost.precision %d, using 2", c.Cost.Precision))
		c.Cost.Precision = 2
	}

	if c.Cache.SparklineWidth <= 0 {
		warn(fmt.Sprintf("invalid cache.sparkline_width %d, using 8", c.Cache.SparklineWidth))
		c.Cache.SparklineWidth = 8
	}

	if c.Git.CacheTTL < 0 {
		warn(fmt.Sprintf("invalid git.cache_ttl %d, using 5", c.Git.CacheTTL))
		c.Git.CacheTTL = 5
	}

	if c.Session.MaxHistory <= 0 {
		warn(fmt.Sprintf("invalid session.max_history %d, using 50", c.Session.MaxHistory))
		c.Session.MaxHistory = 50
	}
}

func loadConfig() Config {
	c := defaultConfig()

	path := configPath
	if path == "" {
		home, err := userHomeDir()
		if err != nil {
			return c
		}
		path = filepath.Join(home, ".claude", "statusline.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}

	if err := json.Unmarshal(data, &c); err != nil {
		fmt.Fprintf(os.Stderr, "claude-statusline: warning: config parse error: %v\n", err)
	}
	return c
}

var cfg = defaultConfig()
