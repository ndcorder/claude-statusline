package main

import (
	"encoding/json"
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
}

type GitConfig struct {
	Enabled     bool `json:"enabled"`
	AheadBehind bool `json:"ahead_behind"`
	Changes     bool `json:"changes"`
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

	json.Unmarshal(data, &c)
	return c
}

var cfg = defaultConfig()
