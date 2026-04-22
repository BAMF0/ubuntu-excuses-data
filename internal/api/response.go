package api

import (
	"github.com/BAMF0/ubuntu-excuses-data/internal/domain"
)

// MetaResponse describes the dataset and its available filter values.
type MetaResponse struct {
	GeneratedDate   string   `json:"generated_date"`
	TotalSources    int      `json:"total_sources"`
	TotalCandidates int      `json:"total_candidates"`
	Components      []string `json:"components"`
	Verdicts        []string `json:"verdicts"`
	Maintainers     []string `json:"maintainers"`
	Arches          []string `json:"arches"`
	Statuses        []string `json:"statuses"`
}

// SourceListResponse is the paginated envelope for a list of sources.
type SourceListResponse struct {
	GeneratedDate string           `json:"generated_date"`
	Total         int              `json:"total"`
	Offset        int              `json:"offset"`
	Limit         int              `json:"limit"`
	Sources       []SourceResponse `json:"sources"`
}

// SourceResponse is the JSON representation of a single source package.
type SourceResponse struct {
	SourcePackage      string              `json:"source_package"`
	Component          string              `json:"component"`
	Maintainer         string              `json:"maintainer"`
	Verdict            string              `json:"verdict"`
	MigrationStatus    string              `json:"migration_status"`
	OldVersion         string              `json:"old_version"`
	NewVersion         string              `json:"new_version"`
	IsCandidate        bool                `json:"is_candidate"`
	InvalidatedByOther bool                `json:"invalidated_by_other"`
	ItemName           string              `json:"item_name"`
	Excuse             ExcuseResponse      `json:"excuse"`
	PolicyInfo         PolicyInfoResponse  `json:"policy_info"`
	Dependencies       *DependencyResponse `json:"dependencies,omitempty"`
	Hints              []HintResponse      `json:"hints,omitempty"`
	Reason             []string            `json:"reason,omitempty"`
}

// ExcuseResponse is the JSON representation of an excuse.
type ExcuseResponse struct {
	Status string   `json:"status"`
	Detail string   `json:"detail,omitempty"`
	Info   []string `json:"info,omitempty"`
}

// DependencyResponse lists blocking and ordering dependencies.
type DependencyResponse struct {
	BlockedBy    []string `json:"blocked_by,omitempty"`
	MigrateAfter []string `json:"migrate_after,omitempty"`
}

// HintResponse represents a migration hint.
type HintResponse struct {
	From string `json:"from"`
	Type string `json:"type"`
}

// PolicyInfoResponse holds per-policy verdicts.
type PolicyInfoResponse struct {
	Age          AgePolicyResponse          `json:"age"`
	Autopkgtest  AutopkgtestPolicyResponse  `json:"autopkgtest"`
	Block        string                     `json:"block"`
	BlockBugs    string                     `json:"block_bugs"`
	Depends      string                     `json:"depends"`
	Email        string                     `json:"email"`
	Linux        *string                    `json:"linux,omitempty"`
	RcBugs       RcBugsPolicyResponse       `json:"rc_bugs"`
	SourcePPA    string                     `json:"source_ppa"`
	UpdateExcuse UpdateExcusePolicyResponse `json:"update_excuse"`
}

// AgePolicyResponse holds age policy details.
type AgePolicyResponse struct {
	AgeRequirement int     `json:"age_requirement"`
	CurrentAge     float64 `json:"current_age"`
	Verdict        string  `json:"verdict"`
}

// AutopkgtestPolicyResponse holds the overall verdict plus per-package results.
type AutopkgtestPolicyResponse struct {
	Verdict  string                                          `json:"verdict"`
	Packages map[string]map[string]AutopkgtestResultResponse `json:"packages,omitempty"`
}

// AutopkgtestResultResponse holds the outcome for a single arch run.
type AutopkgtestResultResponse struct {
	Status string  `json:"status"`
	LogURL *string `json:"log_url,omitempty"`
	PkgURL *string `json:"pkg_url,omitempty"`
}

// RcBugsPolicyResponse holds RC bug details.
type RcBugsPolicyResponse struct {
	SharedBugs       []int  `json:"shared_bugs,omitempty"`
	UniqueSourceBugs []int  `json:"unique_source_bugs,omitempty"`
	UniqueTargetBugs []int  `json:"unique_target_bugs,omitempty"`
	Verdict          string `json:"verdict"`
}

// UpdateExcusePolicyResponse holds the update excuse verdict and bug IDs.
type UpdateExcusePolicyResponse struct {
	Verdict string           `json:"verdict"`
	Bugs    map[string]int64 `json:"bugs,omitempty"`
}

// NewMetaResponse builds a MetaResponse from a domain.Excuses.
func NewMetaResponse(e *domain.Excuses) MetaResponse {
	return MetaResponse{
		GeneratedDate:   e.GeneratedDate.UTC().Format("2006-01-02T15:04:05Z"),
		TotalSources:    len(e.ByName),
		TotalCandidates: len(e.Candidates),
		Components:      e.Components,
		Verdicts:        e.Verdicts,
		Maintainers:     e.Maintainers,
		Arches:          e.Arches,
		Statuses:        e.Statuses,
	}
}

// NewSourceResponse converts a domain.Source into its JSON DTO, resolving
// all interned IDs using the parent Excuses.
func NewSourceResponse(e *domain.Excuses, s *domain.Source) SourceResponse {
	r := SourceResponse{
		SourcePackage:      s.SourcePackage,
		Component:          e.Components[s.ComponentID],
		Maintainer:         e.Maintainers[s.MaintainerID],
		Verdict:            e.Verdicts[s.VerdictID],
		MigrationStatus:    s.Excuse.Status.String(),
		OldVersion:         s.OldVersion,
		NewVersion:         s.NewVersion,
		IsCandidate:        s.IsCandidate,
		InvalidatedByOther: s.InvalidatedByOther,
		ItemName:           s.ItemName,
		Excuse: ExcuseResponse{
			Status: s.Excuse.Status.String(),
			Detail: s.Excuse.Detail,
			Info:   s.Excuse.Info,
		},
		PolicyInfo: newPolicyInfoResponse(e, &s.PolicyInfo),
		Hints:      newHintResponses(s.Hints),
		Reason:     s.Reason,
	}
	if s.Dependencies != nil {
		r.Dependencies = &DependencyResponse{
			BlockedBy:    s.Dependencies.BlockedBy,
			MigrateAfter: s.Dependencies.MigrateAfter,
		}
	}
	return r
}

func newPolicyInfoResponse(e *domain.Excuses, p *domain.PolicyInfo) PolicyInfoResponse {
	return PolicyInfoResponse{
		Age: AgePolicyResponse{
			AgeRequirement: p.Age.AgeRequirement,
			CurrentAge:     p.Age.CurrentAge,
			Verdict:        p.Age.Verdict,
		},
		Autopkgtest: newAutopkgtestPolicyResponse(e, &p.Autopkgtest),
		Block:       p.Block,
		BlockBugs:   p.BlockBugs,
		Depends:     p.Depends,
		Email:       p.Email,
		Linux:       p.Linux,
		RcBugs: RcBugsPolicyResponse{
			SharedBugs:       p.RcBugs.SharedBugs,
			UniqueSourceBugs: p.RcBugs.UniqueSourceBugs,
			UniqueTargetBugs: p.RcBugs.UniqueTargetBugs,
			Verdict:          p.RcBugs.Verdict,
		},
		SourcePPA: p.SourcePPA,
		UpdateExcuse: UpdateExcusePolicyResponse{
			Verdict: p.UpdateExcuse.Verdict,
			Bugs:    p.UpdateExcuse.Bugs,
		},
	}
}

func newAutopkgtestPolicyResponse(e *domain.Excuses, a *domain.AutopkgtestPolicy) AutopkgtestPolicyResponse {
	r := AutopkgtestPolicyResponse{
		Verdict:  a.Verdict,
		Packages: make(map[string]map[string]AutopkgtestResultResponse, len(a.Packages)),
	}
	for pkg, arches := range a.Packages {
		archMap := make(map[string]AutopkgtestResultResponse, len(arches))
		for archID, res := range arches {
			archMap[e.Arches[archID]] = AutopkgtestResultResponse{
				Status: e.Statuses[res.StatusID],
				LogURL: res.LogURL,
				PkgURL: res.PkgURL,
			}
		}
		r.Packages[pkg] = archMap
	}
	return r
}

func newHintResponses(hints []domain.Hint) []HintResponse {
	if len(hints) == 0 {
		return nil
	}
	out := make([]HintResponse, len(hints))
	for i, h := range hints {
		out[i] = HintResponse{From: h.From, Type: h.Type}
	}
	return out
}
