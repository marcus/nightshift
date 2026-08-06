package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MaintenanceChecker assesses dependency maintenance health via GitHub API.
type MaintenanceChecker struct {
	client    *http.Client
	token     string
	cache     map[string]*repoInfo
	cacheMu   sync.RWMutex
	semaphore chan struct{}
}

type repoInfo struct {
	Archived  bool      `json:"archived"`
	PushedAt  time.Time `json:"pushed_at"`
	Stars     int       `json:"stargazers_count"`
	OpenIssue int       `json:"open_issues_count"`
}

// NewMaintenanceChecker creates a checker with GitHub token and concurrency limit.
func NewMaintenanceChecker(client *http.Client, token string, concurrency int) *MaintenanceChecker {
	if concurrency <= 0 {
		concurrency = 3
	}
	return &MaintenanceChecker{
		client:    client,
		token:     token,
		cache:     make(map[string]*repoInfo),
		semaphore: make(chan struct{}, concurrency),
	}
}

// CheckMaintenance evaluates maintenance health for all dependencies.
func (mc *MaintenanceChecker) CheckMaintenance(ctx context.Context, deps []Dependency) ([]Finding, error) {
	var mu sync.Mutex
	var findings []Finding
	var wg sync.WaitGroup

	for _, dep := range deps {
		// Only check github.com modules
		if !strings.HasPrefix(dep.Module, "github.com/") {
			continue
		}

		wg.Add(1)
		go func(d Dependency) {
			defer wg.Done()

			select {
			case mc.semaphore <- struct{}{}:
				defer func() { <-mc.semaphore }()
			case <-ctx.Done():
				return
			}

			f := mc.checkRepo(ctx, d)
			if len(f) > 0 {
				mu.Lock()
				findings = append(findings, f...)
				mu.Unlock()
			}
		}(dep)
	}

	wg.Wait()

	if ctx.Err() != nil {
		return findings, ctx.Err()
	}
	return findings, nil
}

func (mc *MaintenanceChecker) checkRepo(ctx context.Context, dep Dependency) []Finding {
	owner, repo := parseGitHubModule(dep.Module)
	if owner == "" || repo == "" {
		return nil
	}

	cacheKey := owner + "/" + repo

	mc.cacheMu.RLock()
	info, cached := mc.cache[cacheKey]
	mc.cacheMu.RUnlock()

	if !cached {
		var err error
		info, err = mc.fetchRepoInfo(ctx, owner, repo)
		if err != nil {
			return nil
		}
		mc.cacheMu.Lock()
		mc.cache[cacheKey] = info
		mc.cacheMu.Unlock()
	}

	var findings []Finding

	if info.Archived {
		findings = append(findings, Finding{
			Module:      dep.Module,
			Version:     dep.Version,
			Category:    CategoryMaintenance,
			Risk:        RiskHigh,
			Title:       "Archived repository",
			Description: fmt.Sprintf("%s/%s is archived and no longer maintained", owner, repo),
			Reference:   fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		})
	}

	// Check last activity (>1 year = medium risk, >2 years = high risk)
	sinceLastPush := time.Since(info.PushedAt)
	if sinceLastPush > 2*365*24*time.Hour {
		findings = append(findings, Finding{
			Module:      dep.Module,
			Version:     dep.Version,
			Category:    CategoryMaintenance,
			Risk:        RiskHigh,
			Title:       "Unmaintained dependency",
			Description: fmt.Sprintf("No activity for %d months", int(sinceLastPush.Hours()/24/30)),
			Reference:   fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		})
	} else if sinceLastPush > 365*24*time.Hour {
		findings = append(findings, Finding{
			Module:      dep.Module,
			Version:     dep.Version,
			Category:    CategoryMaintenance,
			Risk:        RiskMedium,
			Title:       "Stale dependency",
			Description: fmt.Sprintf("No activity for %d months", int(sinceLastPush.Hours()/24/30)),
			Reference:   fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		})
	}

	return findings
}

func (mc *MaintenanceChecker) fetchRepoInfo(ctx context.Context, owner, repo string) (*repoInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if mc.token != "" {
		req.Header.Set("Authorization", "Bearer "+mc.token)
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d for %s/%s", resp.StatusCode, owner, repo)
	}

	var info repoInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// parseGitHubModule extracts owner/repo from a Go module path like github.com/owner/repo/...
func parseGitHubModule(module string) (owner, repo string) {
	parts := strings.Split(module, "/")
	if len(parts) < 3 || parts[0] != "github.com" {
		return "", ""
	}
	return parts[1], parts[2]
}
