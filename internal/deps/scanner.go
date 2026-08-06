package deps

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Logger defines the logging interface used by the scanner.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Scanner orchestrates dependency scanning across projects.
type Scanner struct {
	store       *Store
	githubToken string
	logger      Logger
}

// NewScanner creates a Scanner with the given dependencies.
func NewScanner(store *Store, githubToken string, logger Logger) *Scanner {
	return &Scanner{
		store:       store,
		githubToken: githubToken,
		logger:      logger,
	}
}

// ScanProject scans a single project's dependencies for risks.
// It runs vulnerability, maintenance, and license checks concurrently
// and aggregates the results. Partial results are returned alongside
// any errors encountered during scanning.
func (s *Scanner) ScanProject(ctx context.Context, projectPath string) (*ScanResult, error) {
	project := filepath.Base(projectPath)
	s.logger.Infof("scanning dependencies for %s", project)

	deps, err := ParseGoMod(projectPath)
	if err != nil {
		return nil, fmt.Errorf("parsing go.mod for %s: %w", project, err)
	}

	s.logger.Infof("parsed %d dependencies for %s", len(deps), project)

	type checkResult struct {
		findings []Finding
		err      error
		name     string
	}

	results := make(chan checkResult, 3)

	go func() {
		vc := NewVulnChecker()
		findings, err := vc.CheckVulnerabilities(ctx, deps)
		results <- checkResult{findings: findings, err: err, name: "vulnerability"}
	}()

	go func() {
		mc := NewMaintenanceChecker(s.githubToken)
		findings, err := mc.CheckMaintenance(ctx, deps)
		results <- checkResult{findings: findings, err: err, name: "maintenance"}
	}()

	go func() {
		lc := NewLicenseChecker()
		findings, err := lc.CheckLicenses(ctx, deps)
		results <- checkResult{findings: findings, err: err, name: "license"}
	}()

	var allFindings []Finding
	var errs []string

	for i := 0; i < 3; i++ {
		r := <-results
		if r.err != nil {
			s.logger.Warnf("%s check completed with errors: %v", r.name, r.err)
			errs = append(errs, r.err.Error())
		}
		allFindings = append(allFindings, r.findings...)
	}

	SortFindings(allFindings)

	scanResult := &ScanResult{
		Project:   project,
		ScannedAt: time.Now(),
		Deps:      deps,
		Findings:  allFindings,
	}

	if s.store != nil {
		if _, err := s.store.SaveScan(scanResult); err != nil {
			s.logger.Errorf("failed to save scan results: %v", err)
		}
	}

	if len(errs) > 0 {
		return scanResult, fmt.Errorf("scan completed with errors: %s", strings.Join(errs, "; "))
	}

	return scanResult, nil
}

// ScanAll scans multiple projects sequentially.
func (s *Scanner) ScanAll(ctx context.Context, projectPaths []string) ([]*ScanResult, error) {
	var results []*ScanResult
	var errs []string

	for _, path := range projectPaths {
		result, err := s.ScanProject(ctx, path)
		if err != nil {
			s.logger.Warnf("scan for %s completed with errors: %v", path, err)
			errs = append(errs, err.Error())
		}
		if result != nil {
			results = append(results, result)
		}
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("some scans had errors: %s", strings.Join(errs, "; "))
	}
	return results, nil
}
