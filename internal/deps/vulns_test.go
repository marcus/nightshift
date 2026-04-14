package deps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCVSSToRiskLevel(t *testing.T) {
	tests := []struct {
		score float64
		want  RiskLevel
	}{
		{9.8, RiskCritical},
		{9.0, RiskCritical},
		{8.5, RiskHigh},
		{7.0, RiskHigh},
		{5.0, RiskMedium},
		{4.0, RiskMedium},
		{3.9, RiskLow},
		{0.0, RiskLow},
		{-1, RiskMedium},
	}
	for _, tt := range tests {
		got := cvssToRiskLevel(tt.score)
		if got != tt.want {
			t.Errorf("cvssToRiskLevel(%v) = %v, want %v", tt.score, got, tt.want)
		}
	}
}

func TestParseCVSS3Vector(t *testing.T) {
	tests := []struct {
		name   string
		vector string
		want   float64
	}{
		{
			name:   "critical severity",
			vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			want:   9.8,
		},
		{
			name:   "high severity",
			vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
			want:   8.8,
		},
		{
			name:   "medium severity",
			vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:L/I:L/A:N",
			want:   4.2,
		},
		{
			name:   "low severity",
			vector: "CVSS:3.1/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N",
			want:   1.8,
		},
		{
			name:   "empty",
			vector: "",
			want:   -1,
		},
		{
			name:   "incomplete vector",
			vector: "CVSS:3.1/AV:N",
			want:   -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCVSS3Vector(tt.vector)
			if got != tt.want {
				t.Errorf("parseCVSS3Vector(%q) = %v, want %v", tt.vector, got, tt.want)
			}
		})
	}
}

func TestCheckVulnerabilities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var query osvQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if query.Package.Name == "github.com/vuln/pkg" {
			resp := osvResponse{
				Vulns: []osvVuln{
					{
						ID:      "GHSA-1234",
						Aliases: []string{"CVE-2024-1234"},
						Summary: "Remote code execution",
						Severity: []osvSeverity{
							{Type: "CVSS_V3", Score: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		json.NewEncoder(w).Encode(osvResponse{})
	}))
	defer srv.Close()

	vc := NewVulnChecker()
	vc.baseURL = srv.URL

	deps := []Dependency{
		{Module: "github.com/vuln/pkg", Version: "v1.0.0"},
		{Module: "github.com/safe/pkg", Version: "v2.0.0"},
	}

	findings, err := vc.CheckVulnerabilities(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}

	f := findings[0]
	if f.CVEID != "CVE-2024-1234" {
		t.Errorf("CVEID = %q, want CVE-2024-1234", f.CVEID)
	}
	if f.CVSSScore != 9.8 {
		t.Errorf("CVSSScore = %v, want 9.8", f.CVSSScore)
	}
	if f.RiskLevel != RiskCritical {
		t.Errorf("RiskLevel = %v, want critical", f.RiskLevel)
	}
}

func TestCheckVulnerabilitiesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	vc := NewVulnChecker()
	vc.baseURL = srv.URL

	deps := []Dependency{
		{Module: "github.com/some/pkg", Version: "v1.0.0"},
	}

	findings, err := vc.CheckVulnerabilities(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings on error, got %d", len(findings))
	}
}

func TestExtractCVEID(t *testing.T) {
	tests := []struct {
		name    string
		vuln    osvVuln
		wantCVE string
	}{
		{
			name:    "has CVE alias",
			vuln:    osvVuln{ID: "GHSA-1234", Aliases: []string{"CVE-2024-5678"}},
			wantCVE: "CVE-2024-5678",
		},
		{
			name:    "no CVE alias",
			vuln:    osvVuln{ID: "GHSA-1234", Aliases: []string{"GHSA-abcd"}},
			wantCVE: "GHSA-1234",
		},
		{
			name:    "no aliases",
			vuln:    osvVuln{ID: "GO-2024-001"},
			wantCVE: "GO-2024-001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCVEID(tt.vuln)
			if got != tt.wantCVE {
				t.Errorf("extractCVEID() = %q, want %q", got, tt.wantCVE)
			}
		})
	}
}
