package main

type Input struct {
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	Cwd   string `json:"cwd"`
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow ContextWindow `json:"context_window"`
	Cost          Cost          `json:"cost"`
	RateLimits    RateLimits    `json:"rate_limits"`
}

type ContextWindow struct {
	RemainingPercentage *float64     `json:"remaining_percentage"`
	TotalInputTokens    *float64     `json:"total_input_tokens"`
	TotalOutputTokens   *float64     `json:"total_output_tokens"`
	ContextWindowSize   *float64     `json:"context_window_size"`
	CurrentUsage        CurrentUsage `json:"current_usage"`
}

type CurrentUsage struct {
	CacheReadInputTokens     *float64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens *float64 `json:"cache_creation_input_tokens"`
	InputTokens              *float64 `json:"input_tokens"`
}

type Cost struct {
	TotalCostUSD       *float64 `json:"total_cost_usd"`
	TotalDurationMS    *float64 `json:"total_duration_ms"`
	TotalLinesAdded    *float64 `json:"total_lines_added"`
	TotalLinesRemoved  *float64 `json:"total_lines_removed"`
	TotalAPIDurationMS *float64 `json:"total_api_duration_ms"`
}

type RateLimits struct {
	FiveHour struct {
		UsedPercentage *float64 `json:"used_percentage"`
		ResetsAt       *float64 `json:"resets_at"`
	} `json:"five_hour"`
	SevenDay struct {
		UsedPercentage *float64 `json:"used_percentage"`
	} `json:"seven_day"`
}
