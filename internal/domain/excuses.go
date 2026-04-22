package domain

import (
	"fmt"
	"strings"
	"time"
)

// ID types for interned string categories.
// Using distinct named types prevents accidentally mixing up IDs at compile time.
type ComponentID int
type VerdictID int
type MaintainerID int
type StatusID int
type ArchID int
type SwiftAuthID int

// Excuses is the high-performance domain model of update_excuses.yaml.
//
// Categorical strings that recur across thousands of entries (component, verdict,
// maintainer, autopkgtest status, architecture) are stored exactly once in intern
// tables; Sources and test results reference them by small integer ID, reducing
// allocations and enabling O(1) filtered lookups via the secondary index maps.
//
// Construct with NewBuilder.
type Excuses struct {
	GeneratedDate time.Time

	// Release is the Ubuntu release codename (e.g. "resolute"), extracted
	// from autopkgtest URLs during ingest. Used to reconstruct full URLs.
	Release string

	// Intern tables: ID → canonical string value, O(1) lookup by ID.
	Components  []string // indexed by ComponentID
	Verdicts    []string // indexed by VerdictID
	Maintainers []string // indexed by MaintainerID
	Statuses    []string // indexed by StatusID      (e.g. "PASS", "REGRESSION")
	Arches      []string // indexed by ArchID        (e.g. "amd64", "arm64")
	SwiftAuths  []string // indexed by SwiftAuthID   (e.g. "AUTH_0f9aae918d5b4744bf7b827671c86842")

	// Reverse intern tables: canonical string → ID, O(1) lookup by name.
	ComponentIDs  map[string]ComponentID
	VerdictIDs    map[string]VerdictID
	MaintainerIDs map[string]MaintainerID
	StatusIDs     map[string]StatusID
	ArchIDs       map[string]ArchID
	SwiftAuthIDs  map[string]SwiftAuthID

	// ByName provides O(1) lookup of any source by its package name.
	ByName map[string]*Source

	// Secondary index maps for O(1) filtered queries, keyed by interned ID.
	ByComponent  map[ComponentID][]*Source
	ByVerdict    map[VerdictID][]*Source
	ByMaintainer map[MaintainerID][]*Source

	// Candidates is pre-filtered: sources where IsCandidate == true.
	Candidates []*Source
}

// MigrationStatus represents the high-level migration state of a source package.
type MigrationStatus int

const (
	// StatusUnknown indicates the migration status could not be determined.
	StatusUnknown MigrationStatus = iota
	// StatusBlocked means the package is blocked from migrating.
	StatusBlocked
	// StatusWillAttempt means the package will attempt migration.
	StatusWillAttempt
	// StatusWaiting means the package is waiting for test results or another condition.
	StatusWaiting
)

func (s MigrationStatus) String() string {
	switch s {
	case StatusBlocked:
		return "BLOCKED"
	case StatusWillAttempt:
		return "WILL_ATTEMPT"
	case StatusWaiting:
		return "WAITING"
	default:
		return "UNKNOWN"
	}
}

// Excuse holds the parsed migration status for a source package.
// The Status and Detail fields are extracted from the first "Migration status
// for …" excuse line; Info contains the remaining informational entries.
type Excuse struct {
	Status MigrationStatus
	Detail string   // reason text after the status (e.g. "Rejected/violates migration policy/introduces a regression")
	Info   []string // remaining excuse lines (e.g. age info, autopkgtest details)
}

// Source is the domain representation of a single package migration entry.
// Repeated categorical strings are stored as IDs into the parent Excuses
// intern tables rather than as duplicate string values.
type Source struct {
	// Interned IDs — resolve via the parent Excuses intern tables.
	ComponentID  ComponentID
	MaintainerID MaintainerID
	VerdictID    VerdictID

	Dependencies       *Dependencies
	Excuse             Excuse
	Hints              []Hint
	InvalidatedByOther bool
	IsCandidate        bool
	ItemName           string
	NewVersion         string
	OldVersion         string
	PolicyInfo         PolicyInfo
	Reason             []string
	SourcePackage      string
}

// Dependencies lists packages this source must wait for or is blocked by.
type Dependencies struct {
	BlockedBy    []string
	MigrateAfter []string
}

// Hint represents a migration hint (e.g. block, unblock) applied to a source.
type Hint struct {
	From string
	Type string
}

// PolicyInfo holds the per-policy verdicts and details for a source.
type PolicyInfo struct {
	Age          AgePolicy
	Autopkgtest  AutopkgtestPolicy
	Block        string
	BlockBugs    string
	Depends      string
	Email        string
	Linux        *string
	RcBugs       RcBugsPolicy
	SourcePPA    string
	UpdateExcuse UpdateExcusePolicy
}

// AgePolicy describes the age requirement and current age of a source.
type AgePolicy struct {
	AgeRequirement int
	CurrentAge     float64
	Verdict        string
}

// AutopkgtestPolicy holds the overall verdict plus per-package/arch results.
// Packages maps "source/version" → ArchID → AutopkgtestResult for O(1) arch-level
// access. Arch keys and result statuses reference the parent Excuses intern tables.
type AutopkgtestPolicy struct {
	Verdict  string
	Packages map[string]map[ArchID]AutopkgtestResult
}

// AutopkgtestResult holds the outcome for a single arch run.
// StatusID references Excuses.Statuses for the canonical status string.
// LogRunID is the unique run identifier extracted from the full log URL;
// SwiftAuthID references the Swift AUTH token in Excuses.SwiftAuths;
// together they allow on-demand reconstruction via Excuses.LogURL.
type AutopkgtestResult struct {
	StatusID    StatusID
	SwiftAuthID SwiftAuthID
	LogRunID    string // e.g. "20260420_114305_a5f93" (empty when no log available)
}

// RcBugsPolicy lists RC bugs shared or unique to the source/target.
type RcBugsPolicy struct {
	SharedBugs       []int
	UniqueSourceBugs []int
	UniqueTargetBugs []int
	Verdict          string
}

// UpdateExcusePolicy holds the verdict and Launchpad bug IDs blocking migration.
// Bugs maps Launchpad bug ID → last-updated Unix timestamp.
type UpdateExcusePolicy struct {
	Verdict string
	Bugs    map[string]int64
}

// Builder constructs an Excuses incrementally, interning repeated strings as
// sources are added. Call Build to obtain the final Excuses value, which
// callers should treat as read-only by convention and not modify.
type Builder struct {
	e           Excuses
	components  internTable[ComponentID]
	verdicts    internTable[VerdictID]
	maintainers internTable[MaintainerID]
	statuses    internTable[StatusID]
	arches      internTable[ArchID]
	swiftAuths  internTable[SwiftAuthID]
}

// NewBuilder returns a ready-to-use Builder.
func NewBuilder(capacity int) *Builder {
	return &Builder{
		e: Excuses{
			ByName:       make(map[string]*Source, capacity),
			ByComponent:  make(map[ComponentID][]*Source),
			ByVerdict:    make(map[VerdictID][]*Source),
			ByMaintainer: make(map[MaintainerID][]*Source),
		},
		components:  newInternTable[ComponentID](),
		verdicts:    newInternTable[VerdictID](),
		maintainers: newInternTable[MaintainerID](),
		statuses:    newInternTable[StatusID](),
		arches:      newInternTable[ArchID](),
		swiftAuths:  newInternTable[SwiftAuthID](),
	}
}

// InternComponent returns the ComponentID for the given string, registering it
// in the intern table if it has not been seen before.
func (b *Builder) InternComponent(s string) ComponentID { return b.components.intern(s) }

// InternVerdict returns the VerdictID for the given string.
func (b *Builder) InternVerdict(s string) VerdictID { return b.verdicts.intern(s) }

// InternMaintainer returns the MaintainerID for the given string.
func (b *Builder) InternMaintainer(s string) MaintainerID { return b.maintainers.intern(s) }

// InternStatus returns the StatusID for the given autopkgtest status string.
func (b *Builder) InternStatus(s string) StatusID { return b.statuses.intern(s) }

// InternArch returns the ArchID for the given architecture string.
func (b *Builder) InternArch(s string) ArchID { return b.arches.intern(s) }

// InternSwiftAuth returns the SwiftAuthID for the given Swift AUTH token.
func (b *Builder) InternSwiftAuth(s string) SwiftAuthID { return b.swiftAuths.intern(s) }

// Add registers a source in all indexes. The source's ComponentID, VerdictID,
// and MaintainerID must have been obtained from this Builder's Intern* methods.
func (b *Builder) Add(s *Source) {
	b.e.ByName[s.SourcePackage] = s
	b.e.ByComponent[s.ComponentID] = append(b.e.ByComponent[s.ComponentID], s)
	b.e.ByVerdict[s.VerdictID] = append(b.e.ByVerdict[s.VerdictID], s)
	b.e.ByMaintainer[s.MaintainerID] = append(b.e.ByMaintainer[s.MaintainerID], s)
	if s.IsCandidate {
		b.e.Candidates = append(b.e.Candidates, s)
	}
}

// Build finalises construction and returns the Excuses. The Builder must not
// be used after Build is called.
func (b *Builder) Build(generatedDate time.Time) *Excuses {
	b.e.GeneratedDate = generatedDate
	b.e.Components, b.e.ComponentIDs = b.components.export()
	b.e.Verdicts, b.e.VerdictIDs = b.verdicts.export()
	b.e.Maintainers, b.e.MaintainerIDs = b.maintainers.export()
	b.e.Statuses, b.e.StatusIDs = b.statuses.export()
	b.e.Arches, b.e.ArchIDs = b.arches.export()
	b.e.SwiftAuths, b.e.SwiftAuthIDs = b.swiftAuths.export()
	return &b.e
}

// SetRelease stores the Ubuntu release codename.
func (b *Builder) SetRelease(release string) { b.e.Release = release }

// URL reconstruction constants.
const (
	swiftBase  = "https://objectstorage.prodstack5.canonical.com/swift/v1"
	pkgURLBase = "https://autopkgtest.ubuntu.com/packages"
)

// poolPrefix returns the Debian pool-style prefix for a package name:
// "lib" + 4th char for packages starting with "lib", otherwise the first char.
func poolPrefix(pkg string) string {
	if strings.HasPrefix(pkg, "lib") && len(pkg) > 3 {
		return pkg[:4]
	}
	return pkg[:1]
}

// LogURL reconstructs the full autopkgtest log URL for a result.
// pkg is the source package name (not "pkg/version"), arch is the architecture string.
// Returns empty string when the result has no log run ID.
func (e *Excuses) LogURL(pkg, arch string, r *AutopkgtestResult) string {
	if r.LogRunID == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/autopkgtest-%s/%s/%s/%s/%s/%s@/log.gz",
		swiftBase, e.SwiftAuths[r.SwiftAuthID], e.Release, e.Release, arch, poolPrefix(pkg), pkg, r.LogRunID)
}

// PkgURL reconstructs the autopkgtest package history URL.
// pkg is the source package name, arch is the architecture string.
func (e *Excuses) PkgURL(pkg, arch string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		pkgURLBase, poolPrefix(pkg), pkg, e.Release, arch)
}

// internTable is a generic bidirectional intern table used during construction.
// It is unexported; callers interact with it through Builder.
type internTable[ID ~int] struct {
	ids  map[string]ID
	data []string
}

func newInternTable[ID ~int]() internTable[ID] {
	return internTable[ID]{ids: make(map[string]ID)}
}

func (t *internTable[ID]) intern(s string) ID {
	if id, ok := t.ids[s]; ok {
		return id
	}
	id := ID(len(t.data))
	t.ids[s] = id
	t.data = append(t.data, s)
	return id
}

// export returns the forward slice and reverse map built up so far.
func (t *internTable[ID]) export() ([]string, map[string]ID) {
	out := make([]string, len(t.data))
	copy(out, t.data)
	return out, t.ids
}
