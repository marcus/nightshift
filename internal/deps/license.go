package deps

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Known license risk classifications
var licenseRisk = map[string]RiskLevel{
	"MIT":            RiskNone,
	"Apache-2.0":    RiskNone,
	"BSD-2-Clause":  RiskNone,
	"BSD-3-Clause":  RiskNone,
	"ISC":           RiskNone,
	"MPL-2.0":       RiskLow,
	"LGPL-2.1":      RiskMedium,
	"LGPL-3.0":      RiskMedium,
	"GPL-2.0":       RiskHigh,
	"GPL-3.0":       RiskHigh,
	"AGPL-3.0":      RiskCritical,
	"SSPL-1.0":      RiskCritical,
	"BSL-1.1":       RiskHigh,
	"Unlicense":     RiskNone,
	"CC0-1.0":       RiskNone,
}

// LicenseChecker checks dependency licenses for copyleft or restrictive terms.
type LicenseChecker struct {
	client    *http.Client
	semaphore chan struct{}
}

// NewLicenseChecker creates a license checker with the given concurrency.
func NewLicenseChecker(client *http.Client, concurrency int) *LicenseChecker {
	if concurrency <= 0 {
		concurrency = 5
	}
	return &LicenseChecker{
		client:    client,
		semaphore: make(chan struct{}, concurrency),
	}
}

// CheckLicenses checks all dependencies for license risks.
func (lc *LicenseChecker) CheckLicenses(ctx context.Context, deps []Dependency) ([]Finding, error) {
	var mu sync.Mutex
	var findings []Finding
	var wg sync.WaitGroup

	for _, dep := range deps {
		if !strings.HasPrefix(dep.Module, "github.com/") {
			continue
		}

		wg.Add(1)
		go func(d Dependency) {
			defer wg.Done()

			select {
			case lc.semaphore <- struct{}{}:
				defer func() { <-lc.semaphore }()
			case <-ctx.Done():
				return
			}

			f := lc.checkLicense(ctx, d)
			if f != nil {
				mu.Lock()
				findings = append(findings, *f)
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

func (lc *LicenseChecker) checkLicense(ctx context.Context, dep Dependency) *Finding {
	owner, repo := parseGitHubModule(dep.Module)
	if owner == "" || repo == "" {
		return nil
	}

	// Try fetching LICENSE file from GitHub raw content
	licenseText, err := lc.fetchLicense(ctx, owner, repo)
	if err != nil {
		return nil
	}

	spdxID := detectLicense(licenseText)
	if spdxID == "" {
		return nil
	}

	risk, known := licenseRisk[spdxID]
	if !known || risk == RiskNone {
		return nil
	}

	return &Finding{
		Module:      dep.Module,
		Version:     dep.Version,
		Category:    CategoryLicense,
		Risk:        risk,
		Title:       fmt.Sprintf("Restrictive license: %s", spdxID),
		Description: fmt.Sprintf("%s uses %s which may impose distribution requirements", dep.Module, spdxID),
		Reference:   fmt.Sprintf("https://github.com/%s/%s/blob/main/LICENSE", owner, repo),
	}
}

func (lc *LicenseChecker) fetchLicense(ctx context.Context, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/HEAD/LICENSE", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := lc.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("license fetch returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// detectLicense uses heuristic matching to identify the SPDX license ID.
func detectLicense(text string) string {
	lower := strings.ToLower(text)

	switch {
	case strings.Contains(lower, "gnu affero general public license"):
		return "AGPL-3.0"
	case strings.Contains(lower, "gnu general public license") && strings.Contains(lower, "version 3"):
		return "GPL-3.0"
	case strings.Contains(lower, "gnu general public license") && strings.Contains(lower, "version 2"):
		return "GPL-2.0"
	case strings.Contains(lower, "gnu lesser general public license") && strings.Contains(lower, "version 3"):
		return "LGPL-3.0"
	case strings.Contains(lower, "gnu lesser general public license") && strings.Contains(lower, "version 2"):
		return "LGPL-2.1"
	case strings.Contains(lower, "mozilla public license") && strings.Contains(lower, "2.0"):
		return "MPL-2.0"
	case strings.Contains(lower, "apache license") && strings.Contains(lower, "version 2.0"):
		return "Apache-2.0"
	case strings.Contains(lower, "mit license") || (strings.Contains(lower, "permission is hereby granted, free of charge")):
		return "MIT"
	case strings.Contains(lower, "redistribution and use in source and binary forms"):
		if strings.Contains(lower, "neither the name") {
			return "BSD-3-Clause"
		}
		return "BSD-2-Clause"
	case strings.Contains(lower, "server side public license"):
		return "SSPL-1.0"
	case strings.Contains(lower, "business source license"):
		return "BSL-1.1"
	case strings.Contains(lower, "unlicense"):
		return "Unlicense"
	case strings.Contains(lower, "isc license"):
		return "ISC"
	}

	return ""
}
