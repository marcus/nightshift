package deps

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LicenseChecker detects license types for dependencies.
type LicenseChecker struct {
	client *http.Client
	sem    chan struct{}
}

// NewLicenseChecker creates a LicenseChecker with sensible defaults.
func NewLicenseChecker() *LicenseChecker {
	return &LicenseChecker{
		client: &http.Client{Timeout: 10 * time.Second},
		sem:    make(chan struct{}, 10),
	}
}

// CheckLicenses checks the license of each dependency.
// Non-GitHub modules are marked as unknown (Medium risk).
// Returns an error if any license fetch fails.
func (lc *LicenseChecker) CheckLicenses(ctx context.Context, deps []Dependency) ([]Finding, error) {
	var (
		mu       sync.Mutex
		findings []Finding
		errs     []string
	)

	var wg sync.WaitGroup
	for _, dep := range deps {
		owner, repo := parseGitHubModule(dep.Module)
		if owner == "" {
			mu.Lock()
			findings = append(findings, Finding{
				Module:    dep.Module,
				Version:   dep.Version,
				RiskLevel: RiskMedium,
				Category:  CategoryLicense,
				Detail:    fmt.Sprintf("%s: unable to determine license (non-GitHub module)", dep.Module),
			})
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(d Dependency, owner, repo string) {
			defer wg.Done()
			lc.sem <- struct{}{}
			defer func() { <-lc.sem }()

			f, err := lc.checkLicense(ctx, d, owner, repo)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", d.Module, err))
				return
			}
			if f != nil {
				findings = append(findings, *f)
			}
		}(dep, owner, repo)
	}
	wg.Wait()

	if len(errs) > 0 {
		return findings, fmt.Errorf("license check errors: %s", strings.Join(errs, "; "))
	}
	return findings, nil
}

func (lc *LicenseChecker) checkLicense(ctx context.Context, dep Dependency, owner, repo string) (*Finding, error) {
	content, err := lc.fetchLicenseContent(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	license, risk := classifyLicense(content)
	if risk == RiskNone {
		return nil, nil
	}

	return &Finding{
		Module:    dep.Module,
		Version:   dep.Version,
		RiskLevel: risk,
		Category:  CategoryLicense,
		Detail:    fmt.Sprintf("%s: %s license", dep.Module, license),
	}, nil
}

func (lc *LicenseChecker) fetchLicenseContent(ctx context.Context, owner, repo string) (string, error) {
	for _, filename := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "LICENCE", "COPYING"} {
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/HEAD/%s", owner, repo, filename)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}

		resp, err := lc.client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
			if err != nil {
				continue
			}
			return string(body), nil
		}
	}
	return "", fmt.Errorf("no license file found for %s/%s", owner, repo)
}

// classifyLicense identifies the license type from file content using keyword matching.
func classifyLicense(content string) (string, RiskLevel) {
	upper := strings.ToUpper(content)

	switch {
	case strings.Contains(upper, "GNU AFFERO GENERAL PUBLIC LICENSE"):
		return "AGPL", RiskHigh
	case strings.Contains(upper, "GNU GENERAL PUBLIC LICENSE"):
		if strings.Contains(upper, "LESSER") {
			return "LGPL", RiskMedium
		}
		return "GPL", RiskHigh
	case strings.Contains(upper, "MOZILLA PUBLIC LICENSE"):
		return "MPL", RiskMedium
	case strings.Contains(upper, "MIT LICENSE") || strings.Contains(upper, "PERMISSION IS HEREBY GRANTED, FREE OF CHARGE"):
		return "MIT", RiskNone
	case strings.Contains(upper, "APACHE LICENSE"):
		return "Apache-2.0", RiskNone
	case strings.Contains(upper, "BSD ") && (strings.Contains(upper, "REDISTRIBUTION AND USE") || strings.Contains(upper, "REDISTRIBUTIONS")):
		return "BSD", RiskNone
	case strings.Contains(upper, "ISC LICENSE"):
		return "ISC", RiskNone
	case strings.Contains(upper, "THE UNLICENSE") || strings.Contains(upper, "UNLICENSE"):
		return "Unlicense", RiskNone
	case strings.Contains(upper, "CREATIVE COMMONS"):
		return "CC", RiskMedium
	default:
		return "Unknown", RiskMedium
	}
}
