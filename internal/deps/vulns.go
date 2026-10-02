package deps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// VulnChecker queries the OSV.dev API for known vulnerabilities.
type VulnChecker struct {
	client  *http.Client
	baseURL string
	sem     chan struct{}
}

// NewVulnChecker creates a VulnChecker with sensible defaults.
func NewVulnChecker() *VulnChecker {
	return &VulnChecker{
		client:  &http.Client{Timeout: 10 * time.Second},
		baseURL: "https://api.osv.dev",
		sem:     make(chan struct{}, 10),
	}
}

type osvQuery struct {
	Package osvPackage `json:"package"`
	Version string     `json:"version"`
}

type osvPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type osvResponse struct {
	Vulns []osvVuln `json:"vulns"`
}

type osvVuln struct {
	ID       string        `json:"id"`
	Aliases  []string      `json:"aliases"`
	Summary  string        `json:"summary"`
	Severity []osvSeverity `json:"severity"`
}

type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// CheckVulnerabilities queries OSV.dev for each dependency and returns findings.
// Returns a non-nil error if any API calls fail, along with whatever findings
// were successfully retrieved.
func (vc *VulnChecker) CheckVulnerabilities(ctx context.Context, deps []Dependency) ([]Finding, error) {
	var (
		mu       sync.Mutex
		findings []Finding
		errs     []string
	)

	var wg sync.WaitGroup
	for _, dep := range deps {
		wg.Add(1)
		go func(d Dependency) {
			defer wg.Done()
			vc.sem <- struct{}{}
			defer func() { <-vc.sem }()

			fs, err := vc.queryModule(ctx, d)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s@%s: %v", d.Module, d.Version, err))
				return
			}
			findings = append(findings, fs...)
		}(dep)
	}
	wg.Wait()

	if len(errs) > 0 {
		return findings, fmt.Errorf("vulnerability check errors: %s", strings.Join(errs, "; "))
	}
	return findings, nil
}

func (vc *VulnChecker) queryModule(ctx context.Context, dep Dependency) ([]Finding, error) {
	query := osvQuery{
		Package: osvPackage{Name: dep.Module, Ecosystem: "Go"},
		Version: dep.Version,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshaling query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, vc.baseURL+"/v1/query", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := vc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying OSV.dev: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSV.dev returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result osvResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var findings []Finding
	for _, vuln := range result.Vulns {
		cveID := extractCVEID(vuln)
		cvss := extractCVSSScore(vuln)
		findings = append(findings, Finding{
			Module:    dep.Module,
			Version:   dep.Version,
			RiskLevel: cvssToRiskLevel(cvss),
			Category:  CategoryVulnerability,
			Detail:    formatVulnDetail(vuln, cveID, cvss),
			CVEID:     cveID,
			CVSSScore: cvss,
		})
	}

	return findings, nil
}

func extractCVEID(vuln osvVuln) string {
	for _, alias := range vuln.Aliases {
		if strings.HasPrefix(alias, "CVE-") {
			return alias
		}
	}
	return vuln.ID
}

// extractCVSSScore extracts the CVSS score from a vulnerability's severity data.
// OSV.dev returns CVSS vector strings, not raw numeric scores. We parse the
// vector and compute the base score per the CVSS 3.1 specification.
// Returns -1 if no severity data is available.
func extractCVSSScore(vuln osvVuln) float64 {
	for _, sev := range vuln.Severity {
		if sev.Type == "CVSS_V3" {
			score := parseCVSS3Vector(sev.Score)
			if score >= 0 {
				return score
			}
		}
	}
	return -1
}

// parseCVSS3Vector computes a CVSS 3.1 base score from a vector string.
// Example input: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
func parseCVSS3Vector(vector string) float64 {
	if vector == "" {
		return -1
	}

	metrics := make(map[string]string)
	for _, segment := range strings.Split(vector, "/") {
		kv := strings.SplitN(segment, ":", 2)
		if len(kv) == 2 {
			metrics[kv[0]] = kv[1]
		}
	}

	av, acv, prv, uiv := metrics["AV"], metrics["AC"], metrics["PR"], metrics["UI"]
	cv, iv, avail := metrics["C"], metrics["I"], metrics["A"]
	sv := metrics["S"]

	if av == "" || acv == "" || prv == "" || uiv == "" || cv == "" || iv == "" || avail == "" || sv == "" {
		return -1
	}

	scopeChanged := sv == "C"

	iss := 1 - (1-impactWeight(cv))*(1-impactWeight(iv))*(1-impactWeight(avail))

	var impact float64
	if scopeChanged {
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss*0.9731-0.02, 13)
	} else {
		impact = 6.42 * iss
	}

	if impact <= 0 {
		return 0.0
	}

	exploitability := 8.22 * avWeight(av) * acWeight(acv) * prWeight(prv, scopeChanged) * uiWeight(uiv)

	var base float64
	if scopeChanged {
		base = math.Min(1.08*(impact+exploitability), 10.0)
	} else {
		base = math.Min(impact+exploitability, 10.0)
	}

	return cvssRoundUp(base)
}

func avWeight(v string) float64 {
	switch v {
	case "N":
		return 0.85
	case "A":
		return 0.62
	case "L":
		return 0.55
	case "P":
		return 0.20
	default:
		return 0.85
	}
}

func acWeight(v string) float64 {
	switch v {
	case "L":
		return 0.77
	case "H":
		return 0.44
	default:
		return 0.77
	}
}

func prWeight(v string, scopeChanged bool) float64 {
	switch v {
	case "N":
		return 0.85
	case "L":
		if scopeChanged {
			return 0.68
		}
		return 0.62
	case "H":
		if scopeChanged {
			return 0.50
		}
		return 0.27
	default:
		return 0.85
	}
}

func uiWeight(v string) float64 {
	switch v {
	case "N":
		return 0.85
	case "R":
		return 0.62
	default:
		return 0.85
	}
}

func impactWeight(v string) float64 {
	switch v {
	case "H":
		return 0.56
	case "L":
		return 0.22
	case "N":
		return 0.0
	default:
		return 0.56
	}
}

// cvssRoundUp rounds up to one decimal place per the CVSS spec.
func cvssRoundUp(x float64) float64 {
	i := math.Floor(x * 10)
	if (x*10 - i) > 0.0 {
		return (i + 1) / 10.0
	}
	return i / 10.0
}

func cvssToRiskLevel(score float64) RiskLevel {
	switch {
	case score < 0:
		return RiskMedium
	case score >= 9.0:
		return RiskCritical
	case score >= 7.0:
		return RiskHigh
	case score >= 4.0:
		return RiskMedium
	default:
		return RiskLow
	}
}

func formatVulnDetail(vuln osvVuln, cveID string, cvss float64) string {
	summary := vuln.Summary
	if summary == "" {
		summary = "No description available"
	}
	if cvss >= 0 {
		return fmt.Sprintf("%s (CVSS %.1f): %s", cveID, cvss, summary)
	}
	return fmt.Sprintf("%s: %s", cveID, summary)
}
