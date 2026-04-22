package api

import (
	"net/http"
	"strconv"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

// Pagination holds validated offset and limit values.
type Pagination struct {
	Offset int
	Limit  int
}

// ParsePagination extracts offset and limit from query parameters,
// applying defaults and clamping to valid ranges.
func ParsePagination(r *http.Request) Pagination {
	p := Pagination{
		Offset: 0,
		Limit:  defaultLimit,
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = n
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Limit = n
		}
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
	return p
}

// SourceFilters holds the optional filter criteria for listing sources.
type SourceFilters struct {
	Component       string
	Verdict         string
	Maintainer      string
	MigrationStatus string
}

// ParseSourceFilters extracts filter parameters from the request.
func ParseSourceFilters(r *http.Request) SourceFilters {
	q := r.URL.Query()
	return SourceFilters{
		Component:       q.Get("component"),
		Verdict:         q.Get("verdict"),
		Maintainer:      q.Get("maintainer"),
		MigrationStatus: q.Get("status"),
	}
}

// IsEmpty returns true if no filters are set.
func (f SourceFilters) IsEmpty() bool {
	return f.Component == "" && f.Verdict == "" && f.Maintainer == "" && f.MigrationStatus == ""
}
