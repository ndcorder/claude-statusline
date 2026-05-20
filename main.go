package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/user"
	"strings"
	"time"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("claude-statusline " + version)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--init-config" {
		data, _ := json.MarshalIndent(defaultConfig(), "", "  ")
		fmt.Println(string(data))
		return
	}
	run(os.Stdin, os.Stdout)
}

func run(r io.Reader, w io.Writer) {
	cfg = loadConfig()
	cfg.validate(os.Stderr)
	if sessionPath == "" {
		sessionPath = cfg.Session.Path
	}
	if historyPath == "" {
		historyPath = cfg.Session.HistoryPath
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return
	}

	var input Input
	if err := json.Unmarshal(data, &input); err != nil && len(data) > 0 {
		fmt.Fprintf(os.Stderr, "claude-statusline: warning: input parse error: %v\n", err)
	}

	u, _ := user.Current()
	username := u.Username
	hostname, _ := os.Hostname()
	if idx := strings.IndexByte(hostname, '.'); idx >= 0 {
		hostname = hostname[:idx]
	}

	cwd := input.Workspace.CurrentDir
	if cwd == "" {
		cwd = input.Cwd
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	rawCwd := cwd
	if home := os.Getenv("HOME"); home != "" && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}

	model := input.Model.DisplayName
	remaining := deref(input.ContextWindow.RemainingPercentage)
	hasRemaining := input.ContextWindow.RemainingPercentage != nil
	costUSD := deref(input.Cost.TotalCostUSD)
	durationMS := deref(input.Cost.TotalDurationMS)
	totalIn := int64(deref(input.ContextWindow.TotalInputTokens))
	totalOut := int64(deref(input.ContextWindow.TotalOutputTokens))
	ctxSize := int64(deref(input.ContextWindow.ContextWindowSize))
	cacheRead := int64(deref(input.ContextWindow.CurrentUsage.CacheReadInputTokens))
	cacheCreate := int64(deref(input.ContextWindow.CurrentUsage.CacheCreationInputTokens))
	inputTokens := int64(deref(input.ContextWindow.CurrentUsage.InputTokens))
	linesAdded := int64(deref(input.Cost.TotalLinesAdded))
	linesRemoved := int64(deref(input.Cost.TotalLinesRemoved))
	apiMS := int64(deref(input.Cost.TotalAPIDurationMS))

	sess := loadSession()
	if totalIn > 0 {
		sess.update(totalIn, cacheRead, cacheCreate, inputTokens)
		sess.LastCost = costUSD
		sess.LastDur = durationMS
		sess.LastOut = float64(totalOut)
		sess.save()
	}

	fmtTok := func(n int64) string {
		return fmtTokensUnit(n, cfg.Tokens.Format)
	}

	addSep := func(b *strings.Builder) {
		if b.Len() > 0 {
			b.WriteString(Sep)
		}
	}

	// ── LINE 1 ──

	var l1 strings.Builder

	if cfg.UserHost {
		l1.WriteString(Cyan + username + "@" + hostname + Reset)
	}
	if cfg.Cwd {
		if l1.Len() > 0 {
			l1.WriteString(" ")
		}
		l1.WriteString(Yellow + cwd + Reset)
	}

	if cfg.Model && model != "" {
		addSep(&l1)
		l1.WriteString(Magenta + model + Reset)
	}

	if cfg.Git.Enabled {
		if gi := getGitInfo(rawCwd); gi != nil {
			addSep(&l1)
			l1.WriteString(Blue + gi.Branch + Reset)
			if cfg.Git.AheadBehind {
				if gi.Ahead > 0 {
					l1.WriteString(" " + Green + "↑" + itoa(gi.Ahead) + Reset)
				}
				if gi.Behind > 0 {
					l1.WriteString(" " + Red + "↓" + itoa(gi.Behind) + Reset)
				}
			}
			if cfg.Git.Changes && gi.Changed > 0 {
				l1.WriteString(" " + Green + "+" + itoa(gi.Added) + Reset +
					Dim + "/" + Reset + Red + "-" + itoa(gi.Removed) + Reset +
					" " + Dim + "(" + itoa(gi.Changed) + "f)" + Reset)
			}
		}
	}

	if cfg.ContextBar.Enabled && hasRemaining {
		remainInt := int(remaining)
		usedInt := 100 - remainInt
		var ctxColor string
		switch {
		case remainInt <= 10:
			ctxColor = Red
		case remainInt <= 25:
			ctxColor = Yellow
		default:
			ctxColor = Green
		}
		barWidth := cfg.ContextBar.Width
		filled := usedInt * barWidth / 100
		empty := barWidth - filled
		bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
		addSep(&l1)
		l1.WriteString(ctxColor + bar + Reset + " " + Dim + itoa(remainInt) + "%" + Reset)
	}

	if cfg.Cost.Enabled && costUSD > 0 {
		addSep(&l1)
		precision := cfg.Cost.Precision
		fmtStr := fmt.Sprintf("%%.%df", precision)
		l1.WriteString(Dim + "$" + Reset + Yellow + fmt.Sprintf(fmtStr, costUSD) + Reset)
		if cfg.Cost.Velocity && durationMS > 60000 {
			switch cfg.Cost.VelocityUnit {
			case "minute":
				costMin := costUSD * 60000 / durationMS
				l1.WriteString(Dim + fmt.Sprintf("@$%.2f/min", costMin) + Reset)
			default:
				costHr := costUSD * 3600000 / durationMS
				l1.WriteString(Dim + fmt.Sprintf("@$%.2f/hr", costHr) + Reset)
			}
		}
	}

	if cfg.Duration.Enabled && durationMS > 0 {
		addSep(&l1)
		l1.WriteString(Dim + fmtDuration(durationMS, cfg.Duration.Format) + Reset)
	}

	if l1.Len() > 0 {
		fmt.Fprintln(w, l1.String())
	}

	// ── LINE 2 ──

	var parts []string

	if cfg.Tokens.Enabled && totalIn > 0 && totalOut > 0 {
		parts = append(parts, Dim+"tok"+Reset+" "+
			Cyan+"↓"+fmtTok(totalIn)+Reset+" "+
			Magenta+"↑"+fmtTok(totalOut)+Reset)
	}

	if cfg.Cache.Enabled {
		if cfg.Cache.Cumulative {
			sTotalCache := sess.CumCR + sess.CumCC + sess.CumIT
			if sTotalCache > 0 {
				sHitPct := int(sess.CumCR * 100 / sTotalCache)
				sHitColor := colorPct(100 - sHitPct)
				parts = append(parts, Dim+"cache"+Reset+" "+
					sHitColor+itoa(sHitPct)+"%"+Reset+Dim+"hit"+Reset+" "+
					Green+"r:"+fmtTok(sess.CumCR)+Reset+
					Dim+"/"+Reset+
					Yellow+"w:"+fmtTok(sess.CumCC)+Reset)
			}
		} else {
			reqTotal := cacheRead + cacheCreate + inputTokens
			if reqTotal > 0 {
				hitPct := int(cacheRead * 100 / reqTotal)
				hitColor := colorPct(100 - hitPct)
				parts = append(parts, Dim+"cache"+Reset+" "+
					hitColor+itoa(hitPct)+"%"+Reset+Dim+"hit"+Reset+" "+
					Green+"r:"+fmtTok(cacheRead)+Reset+
					Dim+"/"+Reset+
					Yellow+"w:"+fmtTok(cacheCreate)+Reset)
			}
		}
	}

	if cfg.Cache.Enabled && cfg.Cache.Savings {
		cr, cc := sess.CumCR, sess.CumCC
		if !cfg.Cache.Cumulative {
			cr, cc = cacheRead, cacheCreate
		}
		if cr > 0 || cc > 0 {
			var saveRate, overheadRate float64
			modelLower := strings.ToLower(model)
			switch {
			case strings.Contains(modelLower, "opus"):
				saveRate, overheadRate = cfg.Pricing.Opus.CacheReadRate, cfg.Pricing.Opus.CacheCreateRate
			case strings.Contains(modelLower, "haiku"):
				saveRate, overheadRate = cfg.Pricing.Haiku.CacheReadRate, cfg.Pricing.Haiku.CacheCreateRate
			default:
				saveRate, overheadRate = cfg.Pricing.Sonnet.CacheReadRate, cfg.Pricing.Sonnet.CacheCreateRate
			}
			netSavings := (float64(cr)*saveRate - float64(cc)*overheadRate) / 1_000_000
			if math.Abs(netSavings) >= 0.001 {
				display := fmt.Sprintf("%.3f", netSavings)
				display = strings.TrimRight(display, "0")
				if dot := strings.IndexByte(display, '.'); dot >= 0 {
					decimals := len(display) - dot - 1
					if decimals < 2 {
						display += strings.Repeat("0", 2-decimals)
					}
				}
				var savingsColor string
				if netSavings < 0 {
					savingsColor = Red
				} else {
					savingsColor = Green
					display = "+" + display
				}
				parts = append(parts, Dim+"saved"+Reset+" "+savingsColor+"$"+display+Reset)
			}
		}
	}

	if cfg.Cache.Enabled && cfg.Cache.Sparkline {
		sparkWidth := cfg.Cache.SparklineWidth
		hist := sess.SparkHist
		if len(hist) > sparkWidth {
			hist = hist[len(hist)-sparkWidth:]
		}
		if len(hist) >= 2 {
			sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
			var spark strings.Builder
			for _, v := range hist {
				idx := min(max(v*7/100, 0), 7)
				spark.WriteRune(sparkChars[idx])
			}
			parts = append(parts, Green+spark.String()+Reset)
		}
	}

	if cfg.ContextRunway && hasRemaining && ctxSize > 0 && sess.TurnCount > 0 {
		remainInt := int(remaining)
		remainingTokens := ctxSize * int64(remainInt) / 100
		avgGrowth := totalIn / int64(sess.TurnCount)
		if avgGrowth > 0 {
			estTurns := int(remainingTokens / avgGrowth)
			var runwayColor string
			switch {
			case estTurns <= 3:
				runwayColor = Red
			case estTurns <= 8:
				runwayColor = Yellow
			default:
				runwayColor = Green
			}
			parts = append(parts, Dim+"~"+Reset+runwayColor+itoa(estTurns)+Reset+Dim+"turns"+Reset)
		}
	}

	if cfg.RateLimits.Enabled && input.RateLimits.FiveHour.UsedPercentage != nil {
		rl5Int := int(*input.RateLimits.FiveHour.UsedPercentage)
		rl5Color := colorPct(rl5Int)
		rlStr := Dim + "5h" + Reset + rl5Color + itoa(rl5Int) + "%" + Reset

		if cfg.RateLimits.ShowReset && input.RateLimits.FiveHour.ResetsAt != nil {
			resetAt := int64(*input.RateLimits.FiveHour.ResetsAt)
			remainSecs := int(resetAt - time.Now().Unix())
			if remainSecs > 0 {
				var resetStr string
				switch {
				case remainSecs >= 3600:
					resetStr = fmt.Sprintf(" %dh%dm", remainSecs/3600, (remainSecs%3600)/60)
				case remainSecs >= 60:
					resetStr = fmt.Sprintf(" %dm", remainSecs/60)
				default:
					resetStr = fmt.Sprintf(" %ds", remainSecs)
				}
				rlStr += Dim + resetStr + Reset
			}
		}

		if cfg.RateLimits.Show7Day && input.RateLimits.SevenDay.UsedPercentage != nil {
			rl7Int := int(*input.RateLimits.SevenDay.UsedPercentage)
			rl7Color := colorPct(rl7Int)
			rlStr += " " + Dim + "7d" + Reset + rl7Color + itoa(rl7Int) + "%" + Reset
		}

		parts = append(parts, Dim+"rate"+Reset+" "+rlStr)
	}

	if cfg.LineDeltas && (linesAdded > 0 || linesRemoved > 0) {
		parts = append(parts, Dim+"Δ"+Reset+
			Green+"+"+itoa64(linesAdded)+Reset+
			Dim+"/"+Reset+
			Red+"-"+itoa64(linesRemoved)+Reset)
	}

	if cfg.ApiStats.Enabled && apiMS > 0 && durationMS > 0 {
		apiPct := int(apiMS * 100 / int64(durationMS))
		apiSecs := apiMS / 1000
		apiStr := Dim + "api" + Reset + " " + Blue + itoa64(apiSecs) + "s" + Reset +
			Dim + "(" + itoa(apiPct) + "%)" + Reset
		if cfg.ApiStats.Throughput && totalOut > 0 && apiMS > 0 {
			tps := totalOut * 1000 / apiMS
			apiStr += " " + Magenta + itoa64(tps) + "t/s" + Reset
		}
		parts = append(parts, apiStr)
	}

	if cfg.SessionCompare && costUSD > 0 {
		if avg := loadHistoryAvg(); avg != nil && avg.Count > 0 {
			diff := costUSD - avg.AvgCost
			var cmpColor, comparison string
			if diff >= 0 {
				comparison = fmt.Sprintf("+%.2f", diff)
				cmpColor = Yellow
			} else {
				comparison = fmt.Sprintf("%.2f", diff)
				cmpColor = Green
			}
			parts = append(parts, Dim+"vs"+itoa(avg.Count)+"avg"+Reset+" "+cmpColor+"$"+comparison+Reset)
		}
	}

	if len(parts) > 0 {
		var l2 strings.Builder
		for i, p := range parts {
			if i == 0 {
				l2.WriteString("  " + p)
			} else {
				l2.WriteString(Sep + p)
			}
		}
		fmt.Fprintln(w, l2.String())
	}
}
