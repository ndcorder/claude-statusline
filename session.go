package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var (
	sessionPath = "/tmp/claude-statusline-session-go"
	historyPath = "/tmp/claude-statusline-history"
)

const maxHistory = 50

type Session struct {
	PrevTotalIn int64   `json:"prev_total_in"`
	CumCR       int64   `json:"cum_cr"`
	CumCC       int64   `json:"cum_cc"`
	CumIT       int64   `json:"cum_it"`
	TurnCount   int     `json:"turn_count"`
	SparkHist   []int   `json:"spark_hist"`
	LastCost    float64 `json:"last_cost"`
	LastDur     float64 `json:"last_dur"`
	LastOut     float64 `json:"last_out"`
}

func loadSession() Session {
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return Session{}
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}
	}
	return s
}

func (s *Session) save() {
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.WriteFile(sessionPath, data, 0644)
}

func (s *Session) update(totalIn int64, cacheRead, cacheCreate, inputTokens int64) {
	reqTotal := cacheRead + cacheCreate + inputTokens
	reqHit := 0
	if reqTotal > 0 {
		reqHit = int(cacheRead * 100 / reqTotal)
	}

	if totalIn < s.PrevTotalIn {
		if cfg.SessionCompare && s.LastCost != 0 {
			appendHistory(s.LastCost, s.LastDur, s.PrevTotalIn, s.LastOut, s.TurnCount)
		}
		s.CumCR = cacheRead
		s.CumCC = cacheCreate
		s.CumIT = inputTokens
		s.TurnCount = 1
		s.SparkHist = []int{reqHit}
	} else if totalIn > s.PrevTotalIn {
		s.CumCR += cacheRead
		s.CumCC += cacheCreate
		s.CumIT += inputTokens
		s.TurnCount++
		s.SparkHist = append(s.SparkHist, reqHit)
		if len(s.SparkHist) > 8 {
			s.SparkHist = s.SparkHist[len(s.SparkHist)-8:]
		}
	}

	s.PrevTotalIn = totalIn
}

func appendHistory(cost, dur float64, totalIn int64, totalOut float64, turns int) {
	line := fmt.Sprintf("%.4f|%.0f|%d|%.0f|%d\n", cost, dur, totalIn, totalOut, turns)
	f, err := os.OpenFile(historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	f.Close()

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > maxHistory {
		_ = os.WriteFile(historyPath, []byte(strings.Join(lines[len(lines)-maxHistory:], "\n")+"\n"), 0644)
	}
}

type HistoryAvg struct {
	AvgCost float64
	Count   int
}

func loadHistoryAvg() *HistoryAvg {
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil
	}
	var totalCost float64
	var count int
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 1 {
			continue
		}
		var c float64
		if _, err := fmt.Sscanf(parts[0], "%f", &c); err == nil {
			totalCost += c
			count++
		}
	}
	if count == 0 {
		return nil
	}
	return &HistoryAvg{AvgCost: totalCost / float64(count), Count: count}
}
