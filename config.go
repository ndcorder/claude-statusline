package main

type Config struct {
	CostVelocity    bool
	TokenThroughput bool
	CumulativeCache bool
	ContextRunway   bool
	CacheSavings    bool
	Sparkline       bool
	SessionCompare  bool
}

var cfg = Config{
	CostVelocity:    true,
	TokenThroughput: true,
	CumulativeCache: true,
	ContextRunway:   true,
	CacheSavings:    true,
	Sparkline:       true,
	SessionCompare:  true,
}
