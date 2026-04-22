package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/BAMF0/ubuntu-excuses-data/internal/domain"
)

// Handler holds a reference to the read-only domain model and serves API requests.
// Pre-computed fields are populated once at construction time since the dataset
// is immutable after startup.
type Handler struct {
	excuses *domain.Excuses

	// Pre-computed at construction time.
	allSorted    []*domain.Source // alphabetically sorted, computed once
	metaRespJSON []byte           // pre-serialized /meta JSON
}

// NewHandler creates a Handler backed by the given Excuses dataset and
// pre-computes derived data that would otherwise be rebuilt per-request.
func NewHandler(e *domain.Excuses) *Handler {
	h := &Handler{excuses: e}
	h.allSorted = h.computeSortedSources()
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

	sources := h.filteredSources(filters)
	sources = append([]*domain.Source(nil), sources...)
	sortSources(sources, sortOrder)

	total := len(sources)
	start, end := clampRange(page.Offset, page.Limit, total)

	items := make([]SourceResponse, 0, end-start)
	for _, s := range sources[start:end] {
		items = append(items, NewSourceResponse(h.excuses, s))
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
	s, ok := h.excuses.ByName[name]
	if !ok {
		writeError(w, http.StatusNotFound, "source not found: "+name)
		return
	}
	writeJSON(w, http.StatusOK, NewSourceResponse(h.excuses, s))
}

// GetSourceAutopkgtest returns the autopkgtest results for a single source.
func (h *Handler) GetSourceAutopkgtest(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s, ok := h.excuses.ByName[name]
	if !ok {
		writeError(w, http.StatusNotFound, "source not found: "+name)
		return
	}
	writeJSON(w, http.StatusOK, newAutopkgtestPolicyResponse(h.excuses, &s.PolicyInfo.Autopkgtest))
}

// filteredSources returns the set of sources matching the given filters.
// When multiple filters are specified, results must match all of them.
func (h *Handler) filteredSources(f SourceFilters) []*domain.Source {
	if f.IsEmpty() {
		return h.allSourcesSorted()
	}

	// Start with the most selective index-backed filter, then intersect.
	var candidates []*domain.Source
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
		candidates = h.allSourcesSorted()
	}

	// Apply remaining filters linearly.
	var out []*domain.Source
	for _, s := range candidates {
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
		out = append(out, s)
	}
	return out
}

// allSourcesSorted returns the pre-computed alphabetically sorted source list.
func (h *Handler) allSourcesSorted() []*domain.Source {
	return h.allSorted
}

// computeSortedSources builds the sorted slice once at startup.
func (h *Handler) computeSortedSources() []*domain.Source {
	sources := make([]*domain.Source, 0, len(h.excuses.ByName))
	for _, s := range h.excuses.ByName {
		sources = append(sources, s)
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].SourcePackage < sources[j].SourcePackage
	})
	return sources
}

// sortSources sorts sources in-place according to the given SortOrder.
// A secondary sort by name is applied for deterministic ordering when primary
// values are equal.
func sortSources(sources []*domain.Source, o SortOrder) {
	sort.Slice(sources, func(i, j int) bool {
		switch o.Field {
		case SortByAge:
			ai, aj := sources[i].PolicyInfo.Age.CurrentAge, sources[j].PolicyInfo.Age.CurrentAge
			if ai != aj {
				if o.Direction == SortDesc {
					return ai > aj
				}
				return ai < aj
			}
			if o.Direction == SortDesc {
				return sources[i].SourcePackage > sources[j].SourcePackage
			}
			return sources[i].SourcePackage < sources[j].SourcePackage
		default: // SortByName
			if o.Direction == SortDesc {
				return sources[i].SourcePackage > sources[j].SourcePackage
			}
			return sources[i].SourcePackage < sources[j].SourcePackage
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
