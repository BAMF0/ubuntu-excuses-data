package ingest

import (
	"strings"

	"github.com/BAMF0/ubuntu-excuses-data/internal/domain"
	yaml "github.com/BAMF0/ubuntu-excuses-data/internal/ingest/yaml"
)

// ToExcuses converts a decoded ExcusesFile into the optimised domain model,
// interning repeated strings and building all lookup indexes in a single pass.
func ToExcuses(f *yaml.ExcusesFile) *domain.Excuses {
	b := domain.NewBuilder(len(f.Sources))
	for i := range f.Sources {
		b.Add(toSource(b, &f.Sources[i]))
	}
	return b.Build(f.GeneratedDate)
}

func toSource(b *domain.Builder, s *yaml.Source) *domain.Source {
	ds := &domain.Source{
		ComponentID:        b.InternComponent(s.Component),
		MaintainerID:       b.InternMaintainer(s.Maintainer),
		VerdictID:          b.InternVerdict(s.MigrationPolicyVerdict),
		Excuse:             toExcuse(s.Excuses),
		Hints:              toHints(s.Hints),
		InvalidatedByOther: s.InvalidatedByOtherPackage,
		IsCandidate:        s.IsCandidate,
		ItemName:           s.ItemName,
		NewVersion:         s.NewVersion,
		OldVersion:         s.OldVersion,
		PolicyInfo:         toPolicyInfo(b, s.PolicyInfo),
		Reason:             copySlice(s.Reason),
		SourcePackage:      s.SourcePackage,
	}
	if s.Dependencies != nil {
		ds.Dependencies = &domain.Dependencies{
			BlockedBy:    copySlice(s.Dependencies.BlockedBy),
			MigrateAfter: copySlice(s.Dependencies.MigrateAfter),
		}
	}
	return ds
}

func toHints(hints []yaml.Hint) []domain.Hint {
	if len(hints) == 0 {
		return nil
	}
	out := make([]domain.Hint, len(hints))
	for i, h := range hints {
		out[i] = domain.Hint{From: h.HintFrom, Type: h.HintType}
	}
	return out
}

func toPolicyInfo(b *domain.Builder, p yaml.PolicyInfo) domain.PolicyInfo {
	pi := domain.PolicyInfo{
		Age: domain.AgePolicy{
			AgeRequirement: p.Age.AgeRequirement,
			CurrentAge:     p.Age.CurrentAge,
			Verdict:        p.Age.Verdict,
		},
		Autopkgtest: toAutopkgtestPolicy(b, p.Autopkgtest),
		Block:       p.Block.Verdict,
		BlockBugs:   p.BlockBugs.Verdict,
		Depends:     p.Depends.Verdict,
		Email:       p.Email.Verdict,
		RcBugs: domain.RcBugsPolicy{
			SharedBugs:       copySlice(p.RcBugs.SharedBugs),
			UniqueSourceBugs: copySlice(p.RcBugs.UniqueSourceBugs),
			UniqueTargetBugs: copySlice(p.RcBugs.UniqueTargetBugs),
			Verdict:          p.RcBugs.Verdict,
		},
		SourcePPA: p.SourcePPA.Verdict,
		UpdateExcuse: domain.UpdateExcusePolicy{
			Verdict: p.UpdateExcuse.Verdict,
			Bugs:    copyMap(p.UpdateExcuse.Bugs),
		},
	}
	if p.Linux != nil {
		v := p.Linux.Verdict
		pi.Linux = &v
	}
	return pi
}

func toAutopkgtestPolicy(b *domain.Builder, a yaml.AutopkgtestPolicy) domain.AutopkgtestPolicy {
	dp := domain.AutopkgtestPolicy{
		Verdict:  a.Verdict,
		Packages: make(map[string]map[domain.ArchID]domain.AutopkgtestResult, len(a.Packages)),
	}
	for pkg, arches := range a.Packages {
		archMap := make(map[domain.ArchID]domain.AutopkgtestResult, len(arches))
		for arch, res := range arches {
			archMap[b.InternArch(arch)] = domain.AutopkgtestResult{
				StatusID: b.InternStatus(res.Status),
				LogURL:   copyPtr(res.LogURL),
				PkgURL:   copyPtr(res.PkgURL),
			}
		}
		dp.Packages[pkg] = archMap
	}
	return dp
}

// toExcuse parses the raw excuse strings into a structured Excuse.
// The first line is expected to be "Migration status for ... ): <status text>".
// The status text is split on ": " to separate status from detail.
func toExcuse(raw []string) domain.Excuse {
	if len(raw) == 0 {
		return domain.Excuse{}
	}

	var statusText string
	// Find "): " which terminates the "Migration status for pkg (old to new)" prefix.
	if idx := strings.Index(raw[0], "): "); idx != -1 {
		statusText = raw[0][idx+3:]
	} else {
		statusText = raw[0]
	}

	excuse := domain.Excuse{
		Info: copySlice(raw[1:]),
	}

	// Split on ": " to separate status from detail (e.g. "BLOCKED: reason").
	if before, after, ok := strings.Cut(statusText, ": "); ok {
		excuse.Status = parseMigrationStatus(before)
		excuse.Detail = after
	} else {
		excuse.Status = parseMigrationStatus(statusText)
	}

	return excuse
}

func parseMigrationStatus(s string) domain.MigrationStatus {
	switch {
	case strings.HasPrefix(s, "BLOCKED"):
		return domain.StatusBlocked
	case strings.HasPrefix(s, "Will attempt migration"):
		return domain.StatusWillAttempt
	case strings.HasPrefix(s, "Waiting"):
		return domain.StatusWaiting
	default:
		return domain.StatusUnknown
	}
}

// copySlice returns a shallow copy of s with its own backing array,
// breaking any reference to the original. Returns nil for nil/empty input.
func copySlice[T any](s []T) []T {
	if len(s) == 0 {
		return nil
	}
	out := make([]T, len(s))
	copy(out, s)
	return out
}

// copyMap returns a shallow copy of m, breaking any reference to the original.
// Returns nil for nil/empty input.
func copyMap[K comparable, V any](m map[K]V) map[K]V {
	if len(m) == 0 {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// copyPtr returns a pointer to a copy of the value pointed to by p,
// breaking any reference to the original. Returns nil for nil input.
func copyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
