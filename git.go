package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)



type GitInfo struct {
	Branch  string `json:"branch"`
	Ahead   int    `json:"ahead"`
	Behind  int    `json:"behind"`
	Changed int    `json:"changed"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

func getGitInfo(dir string) *GitInfo {
	gitDirOut, err := gitCmd(dir, "rev-parse", "--git-dir")
	if err != nil {
		return nil
	}
	gitDir := strings.TrimSpace(gitDirOut)
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}

	cachePath := filepath.Join(gitDir, ".statusline-cache-go")
	if info, err := os.Stat(cachePath); err == nil {
		if time.Since(info.ModTime()) < time.Duration(cfg.Git.CacheTTL)*time.Second {
			if data, err := os.ReadFile(cachePath); err == nil {
				var gi GitInfo
				if json.Unmarshal(data, &gi) == nil {
					return &gi
				}
			}
		}
	}

	gi := &GitInfo{}

	if branch, err := gitCmd(dir, "branch", "--show-current"); err == nil {
		gi.Branch = strings.TrimSpace(branch)
	}
	if gi.Branch == "" {
		if hash, err := gitCmd(dir, "rev-parse", "--short", "HEAD"); err == nil {
			gi.Branch = strings.TrimSpace(hash)
		}
	}

	if _, err := gitCmd(dir, "rev-parse", "--abbrev-ref", "@{u}"); err == nil {
		if counts, err := gitCmd(dir, "rev-list", "--left-right", "--count", "@{u}...HEAD"); err == nil {
			parts := strings.Fields(strings.TrimSpace(counts))
			if len(parts) == 2 {
				gi.Behind, _ = strconv.Atoi(parts[0])
				gi.Ahead, _ = strconv.Atoi(parts[1])
			}
		}
	}

	if status, err := gitCmd(dir, "status", "--porcelain"); err == nil {
		status = strings.TrimSpace(status)
		if status != "" {
			gi.Changed = len(strings.Split(status, "\n"))
		}
	}

	if gi.Changed > 0 {
		if numstat, err := gitCmd(dir, "diff", "--numstat"); err == nil {
			for _, line := range strings.Split(strings.TrimSpace(numstat), "\n") {
				if line == "" {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					a, _ := strconv.Atoi(fields[0])
					r, _ := strconv.Atoi(fields[1])
					gi.Added += a
					gi.Removed += r
				}
			}
		}
	}

	if data, err := json.Marshal(gi); err == nil {
		_ = os.WriteFile(cachePath, data, 0644)
	}

	return gi
}

func gitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	return string(out), err
}
