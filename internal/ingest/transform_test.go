package ingest

import (
	"bytes"
	_ "embed"
	"strings"
	"testing"

	"github.com/BAMF0/ubuntu-excuses-data/internal/domain"
	yaml "github.com/BAMF0/ubuntu-excuses-data/internal/ingest/yaml"
)

func TestToExcuse(t *testing.T) {
	tests := []struct {
		name       string
		raw        []string
		wantStatus domain.MigrationStatus
		wantDetail string
		wantInfo   int
	}{
		{
			name:       "blocked_with_detail",
			raw:        []string{"Migration status for pkg (1.0 to 2.0): BLOCKED: Rejected/violates migration policy/introduces a regression", "Additional info:", "0 days old"},
			wantStatus: domain.StatusBlocked,
			wantDetail: "Rejected/violates migration policy/introduces a regression",
			wantInfo:   2,
		},
		{
			name:       "will_attempt",
			raw:        []string{"Migration status for pkg (1.0 to 2.0): Will attempt migration (Any information below is purely informational)", "Additional info:"},
			wantStatus: domain.StatusWillAttempt,
			wantDetail: "",
			wantInfo:   1,
		},
		{
			name:       "waiting",
			raw:        []string{"Migration status for pkg (1.0 to 2.0): Waiting for test results, another package or too young (no action required now - check later)"},
			wantStatus: domain.StatusWaiting,
			wantDetail: "",
			wantInfo:   0,
		},
		{
			name:       "empty",
			raw:        nil,
			wantStatus: domain.StatusUnknown,
			wantDetail: "",
			wantInfo:   0,
		},
		{
			name:       "blocked_needs_approval",
			raw:        []string{"Migration status for pkg (- to 1.0): BLOCKED: Needs an approval (either due to a freeze, the source suite or a manual hint)"},
			wantStatus: domain.StatusBlocked,
			wantDetail: "Needs an approval (either due to a freeze, the source suite or a manual hint)",
			wantInfo:   0,
		},
		{
			name:       "epoch_version",
			raw:        []string{"Migration status for pkg (1:2.0-1 to 1:3.0-1): BLOCKED: Cannot migrate due to another item, which is blocked (please check which dependencies are stuck)"},
			wantStatus: domain.StatusBlocked,
			wantDetail: "Cannot migrate due to another item, which is blocked (please check which dependencies are stuck)",
			wantInfo:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toExcuse(tt.raw)
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", got.Status, tt.wantStatus)
			}
			if got.Detail != tt.wantDetail {
				t.Errorf("Detail = %q, want %q", got.Detail, tt.wantDetail)
			}
			if len(got.Info) != tt.wantInfo {
				t.Errorf("len(Info) = %d, want %d", len(got.Info), tt.wantInfo)
			}
		})
	}
}

//go:embed yaml/testdata/base-files_excuses.yaml
var baseFilesYAML []byte

func TestParseLogURL(t *testing.T) {
	tests := []struct {
		name     string
		url      *string
		wantRun  string
		wantAuth string
	}{
		{"nil", nil, "", ""},
		{"normal", strPtr("https://objectstorage.prodstack5.canonical.com/swift/v1/AUTH_0f9aae918d5b4744bf7b827671c86842/autopkgtest-resolute/resolute/amd64/b/bash/20260420_114305_a5f93@/log.gz"), "20260420_114305_a5f93", "AUTH_0f9aae918d5b4744bf7b827671c86842"},
		{"different_auth", strPtr("https://objectstorage.prodstack5.canonical.com/swift/v1/AUTH_abc123/autopkgtest-resolute/resolute/amd64/b/bash/20260420_114305_a5f93@/log.gz"), "20260420_114305_a5f93", "AUTH_abc123"},
		{"no_match", strPtr("https://autopkgtest.ubuntu.com/running"), "", ""},
		{"empty", strPtr(""), "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRun, gotAuth := parseLogURL(tt.url)
			if gotRun != tt.wantRun {
				t.Errorf("runID = %q, want %q", gotRun, tt.wantRun)
			}
			if gotAuth != tt.wantAuth {
				t.Errorf("auth = %q, want %q", gotAuth, tt.wantAuth)
			}
		})
	}
}

func TestDetectRelease(t *testing.T) {
	f, err := yaml.ReadExcusesYAML(bytes.NewReader(baseFilesYAML))
	if err != nil {
		t.Fatal(err)
	}
	got := detectRelease(f)
	if got != "resolute" {
		t.Errorf("detectRelease() = %q, want %q", got, "resolute")
	}
}

func TestURLRoundTrip(t *testing.T) {
	f, err := yaml.ReadExcusesYAML(bytes.NewReader(baseFilesYAML))
	if err != nil {
		t.Fatal(err)
	}
	excuses := ToExcuses(f)

	if excuses.Release != "resolute" {
		t.Fatalf("Release = %q, want %q", excuses.Release, "resolute")
	}

	// Walk all autopkgtest results and verify that reconstructed URLs match
	// the original YAML URLs for entries with normal log URLs.
	src := f.Sources[0]
	domSrc := excuses.ByName[src.SourcePackage]
	if domSrc == nil {
		t.Fatal("source not found in domain model")
	}

	for pkgVer, yamlArches := range src.PolicyInfo.Autopkgtest.Packages {
		pkgName := pkgVer
		if i := strings.Index(pkgVer, "/"); i > 0 {
			pkgName = pkgVer[:i]
		}
		domArches, ok := domSrc.PolicyInfo.Autopkgtest.Packages[pkgVer]
		if !ok {
			t.Errorf("missing package %q in domain model", pkgVer)
			continue
		}
		for archName, yamlRes := range yamlArches {
			archID, ok := excuses.ArchIDs[archName]
			if !ok {
				t.Errorf("unknown arch %q", archName)
				continue
			}
			domRes, ok := domArches[archID]
			if !ok {
				t.Errorf("missing arch %q for %q", archName, pkgVer)
				continue
			}

			// Verify PkgURL reconstruction.
			if yamlRes.PkgURL != nil {
				got := excuses.PkgURL(pkgName, archName)
				if got != *yamlRes.PkgURL {
					t.Errorf("PkgURL(%q, %q) = %q, want %q", pkgName, archName, got, *yamlRes.PkgURL)
				}
			}

			// Verify LogURL reconstruction for normal log URLs (skip non-standard ones like /running).
			if yamlRes.LogURL != nil && strings.Contains(*yamlRes.LogURL, "@/log.gz") {
				got := excuses.LogURL(pkgName, archName, &domRes)
				if got != *yamlRes.LogURL {
					t.Errorf("LogURL(%q, %q) = %q, want %q", pkgName, archName, got, *yamlRes.LogURL)
				}
			}
		}
	}
}

func strPtr(s string) *string { return &s }
