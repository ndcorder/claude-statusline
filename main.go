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

func main() {
	run(os.Stdin, os.Stdout)
}

func run(r io.Reader, w io.Writer) {
	data, err := io.ReadAll(r)
	if err != nil {
		return
	}

	var input Input
	_ = json.Unmarshal(data, &input)

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

	// ── LINE 1 ──

	var l1 strings.Builder
	l1.WriteString(Cyan + username + "@" + hostname + Reset + " " + Yellow + cwd + Reset)

	if model != "" {
		l1.WriteString(Sep + Magenta + model + Reset)
	}

	if gi := getGitInfo(rawCwd); gi != nil {
		l1.WriteString(Sep + Blue + gi.Branch + Reset)
		if gi.Ahead > 0 {
			l1.WriteString(" " + Green + "↑" + itoa(gi.Ahead) + Reset)
		}
		if gi.Behind > 0 {
			l1.WriteString(" " + Red + "↓" + itoa(gi.Behind) + Reset)
		}
		if gi.Changed > 0 {
			l1.WriteString(" " + Green + "+" + itoa(gi.Added) + Reset +
				Dim + "/" + Reset + Red + "-" + itoa(gi.Removed) + Reset +
				" " + Dim + "(" + itoa(gi.Changed) + "f)" + Reset)
		}
	}

	if hasRemaining {
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
		const barWidth = 10
		filled := usedInt * barWidth / 100
		empty := barWidth - filled
		bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
		l1.WriteString(Sep + ctxColor + bar + Reset + " " + Dim + itoa(remainInt) + "%" + Reset)
	}

	if costUSD > 0 {
		l1.WriteString(Sep + Dim + "$" + Reset + Yellow + fmt.Sprintf("%.2f", costUSD) + Reset)
		if cfg.CostVelocity && durationMS > 60000 {
			costHr := costUSD * 3600000 / durationMS
			l1.WriteString(Dim + fmt.Sprintf("@$%.2f/hr", costHr) + Reset)
		}
	}

	if durationMS > 0 {
		totalSecs := int(durationMS) / 1000
		var timeStr string
		switch {
		case totalSecs >= 3600:
			timeStr = fmt.Sprintf("%dh%dm", totalSecs/3600, (totalSecs%3600)/60)
		case totalSecs >= 60:
			timeStr = fmt.Sprintf("%dm%ds", totalSecs/60, totalSecs%60)
		default:
			timeStr = fmt.Sprintf("%ds", totalSecs)
		}
		l1.WriteString(Sep + Dim + timeStr + Reset)
	}

	fmt.Fprintln(w, l1.String())

	// ── LINE 2 ──

	var parts []string

	if totalIn > 0 && totalOut > 0 {
		parts = append(parts, Dim+"tok"+Reset+" "+
			Cyan+"↓"+fmtTokens(totalIn)+Reset+" "+
			Magenta+"↑"+fmtTokens(totalOut)+Reset)
	}

	if cfg.CumulativeCache {
		sTotalCache := sess.CumCR + sess.CumCC + sess.CumIT
		if sTotalCache > 0 {
			sHitPct := int(sess.CumCR * 100 / sTotalCache)
			sHitColor := colorPct(100 - sHitPct)
			parts = append(parts, Dim+"cache"+Reset+" "+
				sHitColor+itoa(sHitPct)+"%"+Reset+Dim+"hit"+Reset+" "+
				Green+"r:"+fmtTokens(sess.CumCR)+Reset+
				Dim+"/"+Reset+
				Yellow+"w:"+fmtTokens(sess.CumCC)+Reset)
		}
	} else {
		reqTotal := cacheRead + cacheCreate + inputTokens
		if reqTotal > 0 {
			hitPct := int(cacheRead * 100 / reqTotal)
			hitColor := colorPct(100 - hitPct)
			parts = append(parts, Dim+"cache"+Reset+" "+
				hitColor+itoa(hitPct)+"%"+Reset+Dim+"hit"+Reset+" "+
				Green+"r:"+fmtTokens(cacheRead)+Reset+
				Dim+"/"+Reset+
				Yellow+"w:"+fmtTokens(cacheCreate)+Reset)
		}
	}

	if cfg.CacheSavings {
		cr, cc := sess.CumCR, sess.CumCC
		if !cfg.CumulativeCache {
			cr, cc = cacheRead, cacheCreate
		}
		if cr > 0 || cc > 0 {
			var saveRate, overheadRate float64
			modelLower := strings.ToLower(model)
			switch {
			case strings.Contains(modelLower, "opus"):
				saveRate, overheadRate = 13.50, 3.75
			case strings.Contains(modelLower, "haiku"):
				saveRate, overheadRate = 0.72, 0.20
			default:
				saveRate, overheadRate = 2.70, 0.75
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

	if cfg.Sparkline && len(sess.SparkHist) >= 2 {
		sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
		var spark strings.Builder
		for _, v := range sess.SparkHist {
			idx := v * 7 / 100
			if idx > 7 {
				idx = 7
			}
			if idx < 0 {
				idx = 0
			}
			spark.WriteRune(sparkChars[idx])
		}
		parts = append(parts, Green+spark.String()+Reset)
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

	if input.RateLimits.FiveHour.UsedPercentage != nil {
		rl5Int := int(*input.RateLimits.FiveHour.UsedPercentage)
		rl5Color := colorPct(rl5Int)
		rlStr := Dim + "5h" + Reset + rl5Color + itoa(rl5Int) + "%" + Reset

		if input.RateLimits.FiveHour.ResetsAt != nil {
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

		if input.RateLimits.SevenDay.UsedPercentage != nil {
			rl7Int := int(*input.RateLimits.SevenDay.UsedPercentage)
			rl7Color := colorPct(rl7Int)
			rlStr += " " + Dim + "7d" + Reset + rl7Color + itoa(rl7Int) + "%" + Reset
		}

		parts = append(parts, Dim+"rate"+Reset+" "+rlStr)
	}

	if linesAdded > 0 || linesRemoved > 0 {
		parts = append(parts, Dim+"Δ"+Reset+
			Green+"+"+itoa64(linesAdded)+Reset+
			Dim+"/"+Reset+
			Red+"-"+itoa64(linesRemoved)+Reset)
	}

	if apiMS > 0 && durationMS > 0 {
		apiPct := int(apiMS * 100 / int64(durationMS))
		apiSecs := apiMS / 1000
		apiStr := Dim + "api" + Reset + " " + Blue + itoa64(apiSecs) + "s" + Reset +
			Dim + "(" + itoa(apiPct) + "%)" + Reset
		if cfg.TokenThroughput && totalOut > 0 && apiMS > 0 {
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
