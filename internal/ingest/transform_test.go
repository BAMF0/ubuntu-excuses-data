package ingest

import (
	"testing"

	"github.com/BAMF0/ubuntu-excuses-data/internal/domain"
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
