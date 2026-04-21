package ingest

import (
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
		Excuses:            s.Excuses,
		Hints:              toHints(s.Hints),
		InvalidatedByOther: s.InvalidatedByOtherPackage,
		IsCandidate:        s.IsCandidate,
		ItemName:           s.ItemName,
		NewVersion:         s.NewVersion,
		OldVersion:         s.OldVersion,
		PolicyInfo:         toPolicyInfo(b, s.PolicyInfo),
		Reason:             s.Reason,
		SourcePackage:      s.SourcePackage,
	}
	if s.Dependencies != nil {
		ds.Dependencies = &domain.Dependencies{
			BlockedBy:    s.Dependencies.BlockedBy,
			MigrateAfter: s.Dependencies.MigrateAfter,
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
			SharedBugs:       p.RcBugs.SharedBugs,
			UniqueSourceBugs: p.RcBugs.UniqueSourceBugs,
			UniqueTargetBugs: p.RcBugs.UniqueTargetBugs,
			Verdict:          p.RcBugs.Verdict,
		},
		SourcePPA: p.SourcePPA.Verdict,
		UpdateExcuse: domain.UpdateExcusePolicy{
			Verdict: p.UpdateExcuse.Verdict,
			Bugs:    p.UpdateExcuse.Bugs,
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
				LogURL:   res.LogURL,
				PkgURL:   res.PkgURL,
			}
		}
		dp.Packages[pkg] = archMap
	}
	return dp
}
