package main

import "fmt"

const (
	Cyan    = "\033[36m"
	Yellow  = "\033[33m"
	Green   = "\033[32m"
	Red     = "\033[31m"
	Magenta = "\033[35m"
	Blue    = "\033[34m"
	Dim     = "\033[2m"
	Bold    = "\033[1m"
	Reset   = "\033[0m"
)

var Sep = " " + Dim + "|" + Reset + " "

func fmtTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%d.%dM", n/1_000_000, (n%1_000_000)/100_000)
	}
	if n >= 10_000 {
		return fmt.Sprintf("%dk", n/1_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%d.%dk", n/1_000, (n%1_000)/100)
	}
	return fmt.Sprintf("%d", n)
}

func colorPct(pct int) string {
	if pct >= 75 {
		return Red
	}
	if pct >= 50 {
		return Yellow
	}
	return Green
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func itoa64(n int64) string {
	return fmt.Sprintf("%d", n)
}

func deref(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func fmtTokensUnit(n int64, format string) string {
	switch format {
	case "raw":
		return fmt.Sprintf("%d", n)
	case "k":
		if n >= 1000 {
			return fmt.Sprintf("%.1fk", float64(n)/1000)
		}
		return fmt.Sprintf("%d", n)
	case "M":
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	default:
		return fmtTokens(n)
	}
}

func fmtDuration(ms float64, format string) string {
	totalSecs := int(ms) / 1000
	switch format {
	case "seconds":
		return fmt.Sprintf("%ds", totalSecs)
	case "minutes":
		return fmt.Sprintf("%.1fm", float64(totalSecs)/60)
	case "hours":
		return fmt.Sprintf("%.2fh", float64(totalSecs)/3600)
	default:
		switch {
		case totalSecs >= 3600:
			return fmt.Sprintf("%dh%dm", totalSecs/3600, (totalSecs%3600)/60)
		case totalSecs >= 60:
			return fmt.Sprintf("%dm%ds", totalSecs/60, totalSecs%60)
		default:
			return fmt.Sprintf("%ds", totalSecs)
		}
	}
}
