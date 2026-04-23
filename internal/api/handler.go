package api

import (
	"cmp"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/BAMF0/ubuntu-excuses-data/internal/domain"
)

// Handler holds a reference to the read-only domain model and serves API requests.
// Pre-computed fields are populated once at construction time since the dataset
// is immutable after startup.
type Handler struct {
	excuses *domain.Excuses
	teams   domain.TeamMappings

	// Pre-computed at construction time.
	allSorted    []domain.SourceIdx // sorted by age ascending with name tiebreak, computed once
	metaRespJSON []byte             // pre-serialized /meta JSON
}

// NewHandler creates a Handler backed by the given Excuses dataset and
// pre-computes derived data that would otherwise be rebuilt per-request.
func NewHandler(e *domain.Excuses, teams domain.TeamMappings) *Handler {
	h := &Handler{excuses: e, teams: teams}
	h.allSorted = h.computeSortedIdxs()
	h.metaRespJSON = mustMarshalJSON(NewMetaResponse(e))
	return h
}

func mustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("pre-marshal JSON: %v", err)
	}
	return b
}

// GetMeta returns dataset metadata and available filter values.
// The response is pre-serialized at startup since the dataset is immutable.
func (h *Handler) GetMeta(w http.ResponseWriter, r *http.Request) {
	writeRawJSON(w, http.StatusOK, h.metaRespJSON)
}

// ListSources returns a paginated, optionally filtered and sorted list of sources.
func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	filters := ParseSourceFilters(r)
	page := ParsePagination(r)
	sortOrder := ParseSortOrder(r)

	isDefaultSort := sortOrder.Field == SortByAge && sortOrder.Direction == SortAsc

	idxs := h.filteredIdxs(filters)

	if !isDefaultSort || !filters.IsEmpty() {
		// filteredIdxs may return shared precomputed slices for both empty and
		// non-empty filters; clone before sorting in-place.
		idxs = slices.Clone(idxs)
		h.sortIdxs(idxs, sortOrder)
	}

	total := len(idxs)
	start, end := clampRange(page.Offset, page.Limit, total)

	items := make([]SourceResponse, 0, end-start)
	for _, idx := range idxs[start:end] {
		items = append(items, NewSourceResponse(h.excuses, h.teams, &h.excuses.Sources[idx]))
	}

	writeJSON(w, http.StatusOK, SourceListResponse{
		GeneratedDate: h.excuses.GeneratedDate.UTC().Format("2006-01-02T15:04:05Z"),
		Total:         total,
		Offset:        page.Offset,
		Limit:         page.Limit,
		Sort:          sortOrder.Field.String(),
		Order:         sortOrder.Direction.String(),
		Sources:       items,
	})
}

// GetSource returns a single source by package name.
func (h *Handler) GetSource(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s := h.excuses.SourceByName(name)
	if s == nil {
		writeError(w, http.StatusNotFound, "source not found: "+name)
		return
	}
	writeJSON(w, http.StatusOK, NewSourceResponse(h.excuses, h.teams, s))
}

// GetSourceAutopkgtest returns the autopkgtest results for a single source.
func (h *Handler) GetSourceAutopkgtest(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s := h.excuses.SourceByName(name)
	if s == nil {
		writeError(w, http.StatusNotFound, "source not found: "+name)
		return
	}
	writeJSON(w, http.StatusOK, newAutopkgtestPolicyResponse(h.excuses, &s.PolicyInfo.Autopkgtest))
}

// ListBlocked returns a paginated list of sources with BLOCKED migration status.
func (h *Handler) ListBlocked(w http.ResponseWriter, r *http.Request) {
	filters := ParseSourceFilters(r)
	page := ParsePagination(r)
	sortOrder := ParseSortOrder(r)

	idxs := h.excuses.ByMigrationStatus[domain.StatusBlocked]
	if idxs == nil {
		idxs = []domain.SourceIdx{}
	}

	// Apply search/depends filters if present.
	var filtered []domain.SourceIdx
	if filters.Search != "" || filters.Depends != "" {
		for _, idx := range idxs {
			s := &h.excuses.Sources[idx]
			if filters.Search != "" && !strings.Contains(s.SourcePackage, filters.Search) {
				continue
			}
			if filters.Depends != "" && !dependsOn(s, filters.Depends) {
				continue
			}
			filtered = append(filtered, idx)
		}
	} else {
		filtered = slices.Clone(idxs)
	}

	h.sortIdxs(filtered, sortOrder)

	total := len(filtered)
	start, end := clampRange(page.Offset, page.Limit, total)

	items := make([]BlockedSourceResponse, 0, end-start)
	for _, idx := range filtered[start:end] {
		items = append(items, NewBlockedSourceResponse(h.excuses, h.teams, &h.excuses.Sources[idx]))
	}

	writeJSON(w, http.StatusOK, BlockedListResponse{
		GeneratedDate: h.excuses.GeneratedDate.UTC().Format("2006-01-02T15:04:05Z"),
		Total:         total,
		Offset:        page.Offset,
		Limit:         page.Limit,
		Sort:          sortOrder.Field.String(),
		Order:         sortOrder.Direction.String(),
		Sources:       items,
	})
}

// filteredIdxs returns the set of source indexes matching the given filters.
// When no filters are specified, returns the pre-sorted index slice directly
// (callers must not modify it). When filters are applied, a new slice is returned.
func (h *Handler) filteredIdxs(f SourceFilters) []domain.SourceIdx {
	if f.IsEmpty() {
		return h.allSorted
	}

	// Start with the most selective index-backed filter, then intersect.
	var candidates []domain.SourceIdx
	switch {
	case f.Component != "":
		id, ok := h.excuses.ComponentIDs[f.Component]
		if !ok {
			return nil
		}
		candidates = h.excuses.ByComponent[id]
	case f.Verdict != "":
		id, ok := h.excuses.VerdictIDs[f.Verdict]
		if !ok {
			return nil
		}
		candidates = h.excuses.ByVerdict[id]
	case f.Maintainer != "":
		id, ok := h.excuses.MaintainerIDs[f.Maintainer]
		if !ok {
			return nil
		}
		candidates = h.excuses.ByMaintainer[id]
	default:
		candidates = h.allSorted
	}

	// Apply remaining filters linearly.
	var out []domain.SourceIdx
	for _, idx := range candidates {
		s := &h.excuses.Sources[idx]
		if f.Component != "" && h.excuses.Components[s.ComponentID] != f.Component {
			continue
		}
		if f.Verdict != "" && h.excuses.Verdicts[s.VerdictID] != f.Verdict {
			continue
		}
		if f.Maintainer != "" && h.excuses.Maintainers[s.MaintainerID] != f.Maintainer {
			continue
		}
		if f.MigrationStatus != "" && s.Excuse.Status.String() != f.MigrationStatus {
			continue
		}
		if f.Search != "" && !strings.Contains(s.SourcePackage, f.Search) {
			continue
		}
		if f.Depends != "" && !dependsOn(s, f.Depends) {
			continue
		}
		out = append(out, idx)
	}
	return out
}

// dependsOn returns true if the source has a dependency relationship
// (blocked_by, blocks, or migrate_after) involving the named package.
func dependsOn(s *domain.Source, name string) bool {
	return slices.Contains(s.Dependencies.BlockedBy, name) ||
		slices.Contains(s.Dependencies.Blocks, name) ||
		slices.Contains(s.Dependencies.MigrateAfter, name)
}

// computeSortedIdxs builds the default-sorted (age ascending, name tiebreak)
// index slice once at startup.
func (h *Handler) computeSortedIdxs() []domain.SourceIdx {
	idxs := make([]domain.SourceIdx, len(h.excuses.Sources))
	for i := range idxs {
		idxs[i] = domain.SourceIdx(i)
	}
	src := h.excuses.Sources
	slices.SortFunc(idxs, func(a, b domain.SourceIdx) int {
		if c := cmp.Compare(src[a].PolicyInfo.Age.CurrentAge, src[b].PolicyInfo.Age.CurrentAge); c != 0 {
			return c
		}
		return cmp.Compare(src[a].SourcePackage, src[b].SourcePackage)
	})
	return idxs
}

// sortIdxs sorts indexes in-place according to the given SortOrder.
// A secondary sort by name is applied for deterministic ordering when primary
// values are equal.
func (h *Handler) sortIdxs(idxs []domain.SourceIdx, o SortOrder) {
	src := h.excuses.Sources
	slices.SortFunc(idxs, func(a, b domain.SourceIdx) int {
		switch o.Field {
		case SortByAge:
			c := cmp.Compare(src[a].PolicyInfo.Age.CurrentAge, src[b].PolicyInfo.Age.CurrentAge)
			if c != 0 {
				if o.Direction == SortDesc {
					return -c
				}
				return c
			}
			c = cmp.Compare(src[a].SourcePackage, src[b].SourcePackage)
			if o.Direction == SortDesc {
				return -c
			}
			return c
		default: // SortByName
			c := cmp.Compare(src[a].SourcePackage, src[b].SourcePackage)
			if o.Direction == SortDesc {
				return -c
			}
			return c
		}
	})
}

// clampRange returns a valid [start, end) range within [0, total).
func clampRange(offset, limit, total int) (int, int) {
	if offset >= total {
		return total, total
	}
	end := min(offset+limit, total)
	return offset, end
}

// errorResponse is the JSON body for error responses.
type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("writeJSON: marshal failed: %v", err)
		writeRawJSON(w, http.StatusInternalServerError, []byte(`{"error":"internal server error"}`))
		return
	}
	writeRawJSON(w, status, data)
}

func writeRawJSON(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		log.Printf("writeRawJSON: write failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
