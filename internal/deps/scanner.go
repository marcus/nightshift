package deps

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
)

// ScanOptions configures a dependency scan.
type ScanOptions struct {
	ProjectPath string
	GitHubToken string
	Concurrency int
}

// Scanner orchestrates dependency risk analysis.
type Scanner struct {
	opts ScanOptions
}

// NewScanner creates a scanner with the given options.
func NewScanner(opts ScanOptions) *Scanner {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 5
	}
	return &Scanner{opts: opts}
}

// Scan runs all checks and returns the combined results.
func (s *Scanner) Scan(ctx context.Context) (*ScanResult, error) {
	start := time.Now()

	deps, err := ParseGoMod(s.opts.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("parsing dependencies: %w", err)
	}

	if len(deps) == 0 {
		return &ScanResult{
			Project:   s.opts.ProjectPath,
			Timestamp: start,
			Duration:  time.Since(start).String(),
			Summary:   make(map[RiskLevel]int),
		}, nil
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// If no token is set, try environment
	token := s.opts.GitHubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	vulnChecker := NewVulnChecker(client, s.opts.Concurrency)
	maintChecker := NewMaintenanceChecker(client, token, s.opts.Concurrency)
	licenseChecker := NewLicenseChecker(client, s.opts.Concurrency)

	var vulnFindings, maintFindings, licenseFindings []Finding

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		vulnFindings, err = vulnChecker.CheckVulnerabilities(gctx, deps)
		return err
	})

	g.Go(func() error {
		var err error
		maintFindings, err = maintChecker.CheckMaintenance(gctx, deps)
		return err
	})

	g.Go(func() error {
		var err error
		licenseFindings, err = licenseChecker.CheckLicenses(gctx, deps)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("scanning: %w", err)
	}

	// Combine all findings
	var allFindings []Finding
	allFindings = append(allFindings, vulnFindings...)
	allFindings = append(allFindings, maintFindings...)
	allFindings = append(allFindings, licenseFindings...)

	SortFindings(allFindings)

	directCount := 0
	for _, d := range deps {
		if d.Direct {
			directCount++
		}
	}

	return &ScanResult{
		Project:    s.opts.ProjectPath,
		Timestamp:  start,
		Duration:   time.Since(start).String(),
		TotalDeps:  len(deps),
		DirectDeps: directCount,
		Findings:   allFindings,
		Summary:    SummarizeResults(allFindings),
	}, nil
}
