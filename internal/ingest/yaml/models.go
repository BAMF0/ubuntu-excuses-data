package ingest

import "time"

// ExcusesFile is the top-level structure of update_excuses.yaml.
type ExcusesFile struct {
	GeneratedDate time.Time `yaml:"generated-date"`
	Sources       []Source  `yaml:"sources"`
}

// Source represents a single package migration entry.
type Source struct {
	Component                 string        `yaml:"component"`
	Dependencies              *Dependencies `yaml:"dependencies"`
	Excuses                   []string      `yaml:"excuses"`
	Hints                     []Hint        `yaml:"hints"`
	InvalidatedByOtherPackage bool          `yaml:"invalidated-by-other-package"`
	IsCandidate               bool          `yaml:"is-candidate"`
	ItemName                  string        `yaml:"item-name"`
	Maintainer                string        `yaml:"maintainer"`
	MigrationPolicyVerdict    string        `yaml:"migration-policy-verdict"`
	NewVersion                string        `yaml:"new-version"`
	OldVersion                string        `yaml:"old-version"`
	PolicyInfo                PolicyInfo    `yaml:"policy_info"`
	Reason                    []string      `yaml:"reason"`
	SourcePackage             string        `yaml:"source"`
}

// Dependencies lists packages this source must wait for or is blocked by.
type Dependencies struct {
	BlockedBy    []string `yaml:"blocked-by"`
	MigrateAfter []string `yaml:"migrate-after"`
}

// Hint represents a migration hint (e.g. unblock, block) applied to a source.
type Hint struct {
	HintFrom string `yaml:"hint-from"`
	HintType string `yaml:"hint-type"`
}

// PolicyInfo holds the per-policy verdicts and details for a source.
type PolicyInfo struct {
	Age          AgePolicy          `yaml:"age"`
	Autopkgtest  AutopkgtestPolicy  `yaml:"autopkgtest"`
	Block        VerdictPolicy      `yaml:"block"`
	BlockBugs    VerdictPolicy      `yaml:"block-bugs"`
	Depends      VerdictPolicy      `yaml:"depends"`
	Email        VerdictPolicy      `yaml:"email"`
	Linux        *VerdictPolicy     `yaml:"linux"`
	RcBugs       RcBugsPolicy       `yaml:"rc-bugs"`
	SourcePPA    VerdictPolicy      `yaml:"source-ppa"`
	UpdateExcuse UpdateExcusePolicy `yaml:"update-excuse"`
}

// VerdictPolicy is a simple policy that only carries a verdict string.
type VerdictPolicy struct {
	Verdict string `yaml:"verdict"`
}

// AgePolicy describes the age requirement and current age of a source.
type AgePolicy struct {
	AgeRequirement int     `yaml:"age-requirement"`
	CurrentAge     float64 `yaml:"current-age"`
	Verdict        string  `yaml:"verdict"`
}

// AutopkgtestPolicy holds the overall autopkgtest verdict plus per-package results.
// The YAML shape mixes a "verdict" key with "source/version" keys, so this type
// uses a custom unmarshaler.
type AutopkgtestPolicy struct {
	Verdict string
	// Packages maps "source/version" → arch → result.
	Packages map[string]map[string]AutopkgtestResult
}

// AutopkgtestResult holds the outcome for a single arch run.
// It corresponds to the 5-element YAML sequence [status, log_url, pkg_url, null, null].
type AutopkgtestResult struct {
	Status string
	LogURL *string
	PkgURL *string
}

// RcBugsPolicy lists RC bugs shared or unique to the source/target.
type RcBugsPolicy struct {
	SharedBugs       []int  `yaml:"shared-bugs"`
	UniqueSourceBugs []int  `yaml:"unique-source-bugs"`
	UniqueTargetBugs []int  `yaml:"unique-target-bugs"`
	Verdict          string `yaml:"verdict"`
}

// UpdateExcusePolicy holds the verdict plus any Launchpad bug IDs that block migration.
// The YAML shape mixes a "verdict" key with bug-ID keys (e.g. "2142117": 1771420300),
// so this type uses a custom unmarshaler.
type UpdateExcusePolicy struct {
	Verdict string
	// Bugs maps Launchpad bug ID → last-updated Unix timestamp.
	Bugs map[string]int64
}
