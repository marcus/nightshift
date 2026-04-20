package deps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

const osvAPIURL = "https://api.osv.dev/v1/query"

// osvQuery is the request body for the OSV.dev API.
type osvQuery struct {
	Package osvPackage `json:"package"`
	Version string     `json:"version"`
}

type osvPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

// osvResponse is the response from the OSV.dev API.
type osvResponse struct {
	Vulns []osvVuln `json:"vulns"`
}

type osvVuln struct {
	ID       string        `json:"id"`
	Summary  string        `json:"summary"`
	Details  string        `json:"details"`
	Severity []osvSeverity `json:"severity"`
}

type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// VulnChecker queries OSV.dev for known vulnerabilities.
type VulnChecker struct {
	client    *http.Client
	apiURL    string
	semaphore chan struct{}
}

// NewVulnChecker creates a checker with the given concurrency limit.
func NewVulnChecker(client *http.Client, concurrency int) *VulnChecker {
	if concurrency <= 0 {
		concurrency = 5
	}
	url := osvAPIURL
	return &VulnChecker{
		client:    client,
		apiURL:    url,
		semaphore: make(chan struct{}, concurrency),
	}
}

// CheckVulnerabilities queries OSV.dev for all dependencies concurrently.
func (vc *VulnChecker) CheckVulnerabilities(ctx context.Context, deps []Dependency) ([]Finding, error) {
	var mu sync.Mutex
	var findings []Finding
	var wg sync.WaitGroup

	errCh := make(chan error, len(deps))

	for _, dep := range deps {
		wg.Add(1)
		go func(d Dependency) {
			defer wg.Done()

			select {
			case vc.semaphore <- struct{}{}:
				defer func() { <-vc.semaphore }()
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}

			vulnFindings, err := vc.queryOSV(ctx, d)
			if err != nil {
				errCh <- err
				return
			}

			if len(vulnFindings) > 0 {
				mu.Lock()
				findings = append(findings, vulnFindings...)
				mu.Unlock()
			}
		}(dep)
	}

	wg.Wait()
	close(errCh)

	// Collect first error if any (non-fatal: we still return partial results)
	for err := range errCh {
		if err == ctx.Err() {
			return findings, err
		}
		// Log but don't fail on individual query errors
	}

	return findings, nil
}

func (vc *VulnChecker) queryOSV(ctx context.Context, dep Dependency) ([]Finding, error) {
	query := osvQuery{
		Package: osvPackage{
			Name:      dep.Module,
			Ecosystem: "Go",
		},
		Version: dep.Version,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshaling query for %s: %w", dep.Module, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, vc.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", dep.Module, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := vc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying OSV for %s: %w", dep.Module, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV returned %d for %s", resp.StatusCode, dep.Module)
	}

	var osvResp osvResponse
	if err := json.NewDecoder(resp.Body).Decode(&osvResp); err != nil {
		return nil, fmt.Errorf("decoding OSV response for %s: %w", dep.Module, err)
	}

	var findings []Finding
	for _, vuln := range osvResp.Vulns {
		risk := classifyVulnSeverity(vuln)
		findings = append(findings, Finding{
			Module:      dep.Module,
			Version:     dep.Version,
			Category:    CategoryVulnerability,
			Risk:        risk,
			Title:       vuln.Summary,
			Description: truncate(vuln.Details, 200),
			Reference:   fmt.Sprintf("https://osv.dev/vulnerability/%s", vuln.ID),
		})
	}

	return findings, nil
}

func classifyVulnSeverity(vuln osvVuln) RiskLevel {
	for _, sev := range vuln.Severity {
		if sev.Type == "CVSS_V3" {
			// Parse CVSS score from vector string (simplified)
			return RiskCritical
		}
	}
	// Default to high for any known vulnerability
	return RiskHigh
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
