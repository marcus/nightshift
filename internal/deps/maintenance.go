package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MaintenanceChecker assesses dependency maintenance health via the GitHub API.
type MaintenanceChecker struct {
	client      *http.Client
	githubToken string
	baseURL     string
	sem         chan struct{}
	cache       sync.Map
}

type repoInfo struct {
	Archived bool      `json:"archived"`
	PushedAt time.Time `json:"pushed_at"`
}

// NewMaintenanceChecker creates a MaintenanceChecker.
func NewMaintenanceChecker(githubToken string) *MaintenanceChecker {
	return &MaintenanceChecker{
		client:      &http.Client{Timeout: 10 * time.Second},
		githubToken: githubToken,
		baseURL:     "https://api.github.com",
		sem:         make(chan struct{}, 5),
	}
}

// CheckMaintenance checks each dependency's maintenance health.
// Non-GitHub modules are skipped. Returns an error if any GitHub API calls fail.
func (mc *MaintenanceChecker) CheckMaintenance(ctx context.Context, deps []Dependency) ([]Finding, error) {
	var (
		mu       sync.Mutex
		findings []Finding
		errs     []string
	)

	var wg sync.WaitGroup
	for _, dep := range deps {
		owner, repo := parseGitHubModule(dep.Module)
		if owner == "" {
			continue
		}

		wg.Add(1)
		go func(d Dependency, owner, repo string) {
			defer wg.Done()
			mc.sem <- struct{}{}
			defer func() { <-mc.sem }()

			info, err := mc.getRepoInfo(ctx, owner, repo)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s/%s: %v", owner, repo, err))
				return
			}

			if info.Archived {
				findings = append(findings, Finding{
					Module:    d.Module,
					Version:   d.Version,
					RiskLevel: RiskCritical,
					Category:  CategoryMaintenance,
					Detail:    fmt.Sprintf("%s/%s is archived", owner, repo),
				})
			}

			age := time.Since(info.PushedAt)
			switch {
			case age > 3*365*24*time.Hour:
				findings = append(findings, Finding{
					Module:    d.Module,
					Version:   d.Version,
					RiskLevel: RiskHigh,
					Category:  CategoryMaintenance,
					Detail:    fmt.Sprintf("%s/%s: last commit %.0f days ago", owner, repo, age.Hours()/24),
				})
			case age > 365*24*time.Hour:
				findings = append(findings, Finding{
					Module:    d.Module,
					Version:   d.Version,
					RiskLevel: RiskMedium,
					Category:  CategoryMaintenance,
					Detail:    fmt.Sprintf("%s/%s: last commit %.0f days ago", owner, repo, age.Hours()/24),
				})
			}
		}(dep, owner, repo)
	}
	wg.Wait()

	if len(errs) > 0 {
		return findings, fmt.Errorf("maintenance check errors: %s", strings.Join(errs, "; "))
	}
	return findings, nil
}

func (mc *MaintenanceChecker) getRepoInfo(ctx context.Context, owner, repo string) (*repoInfo, error) {
	cacheKey := owner + "/" + repo
	if cached, ok := mc.cache.Load(cacheKey); ok {
		return cached.(*repoInfo), nil
	}

	url := fmt.Sprintf("%s/repos/%s/%s", mc.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if mc.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+mc.githubToken)
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub returned %d: %s", resp.StatusCode, string(respBody))
	}

	var info repoInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	mc.cache.Store(cacheKey, &info)
	return &info, nil
}

// parseGitHubModule extracts owner/repo from a Go module path.
// Returns empty strings for non-GitHub modules.
func parseGitHubModule(module string) (string, string) {
	if !strings.HasPrefix(module, "github.com/") {
		return "", ""
	}
	parts := strings.SplitN(module, "/", 4)
	if len(parts) < 3 {
		return "", ""
	}
	return parts[1], parts[2]
}
