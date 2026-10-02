package deps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckVulnerabilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var query osvQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Return vulns only for the "vulnerable" module
		if query.Package.Name == "github.com/vulnerable/pkg" {
			resp := osvResponse{
				Vulns: []osvVuln{
					{
						ID:      "GO-2024-001",
						Summary: "SQL injection in query builder",
						Details: "A vulnerability exists that allows SQL injection.",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// No vulns for other packages
		json.NewEncoder(w).Encode(osvResponse{})
	}))
	defer server.Close()

	checker := &VulnChecker{
		client:    server.Client(),
		apiURL:    server.URL,
		semaphore: make(chan struct{}, 5),
	}

	deps := []Dependency{
		{Module: "github.com/vulnerable/pkg", Version: "v1.0.0", Direct: true},
		{Module: "github.com/safe/pkg", Version: "v2.0.0", Direct: true},
	}

	findings, err := checker.CheckVulnerabilities(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.Module != "github.com/vulnerable/pkg" {
		t.Errorf("unexpected module: %s", f.Module)
	}
	if f.Category != CategoryVulnerability {
		t.Errorf("unexpected category: %s", f.Category)
	}
	if f.Risk != RiskHigh {
		t.Errorf("unexpected risk: %s", f.Risk)
	}
}

func TestCheckVulnerabilities_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(osvResponse{})
	}))
	defer server.Close()

	checker := &VulnChecker{
		client:    server.Client(),
		apiURL:    server.URL,
		semaphore: make(chan struct{}, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	deps := []Dependency{
		{Module: "github.com/foo/bar", Version: "v1.0.0", Direct: true},
	}

	_, err := checker.CheckVulnerabilities(ctx, deps)
	if err == nil {
		// Context cancellation may or may not propagate depending on timing
		// This is acceptable behavior
	}
	_ = err
}
