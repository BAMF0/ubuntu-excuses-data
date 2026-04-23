package api

import (
	"net/http"
	"strconv"
	"strings"
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
	Search          string // substring match on source package name
	Depends         string // match packages that depend on (blocked_by, blocks, migrate_after) the given name
	Team            string // exact match on team name
}

// ParseSourceFilters extracts filter parameters from the request.
func ParseSourceFilters(r *http.Request) SourceFilters {
	q := r.URL.Query()
	return SourceFilters{
		Component:       q.Get("component"),
		Verdict:         q.Get("verdict"),
		Maintainer:      q.Get("maintainer"),
		MigrationStatus: q.Get("status"),
		Search:          q.Get("search"),
		Depends:         q.Get("depends"),
		Team:            q.Get("team"),
	}
}

// IsEmpty returns true if no filters are set.
func (f SourceFilters) IsEmpty() bool {
	return f.Component == "" && f.Verdict == "" && f.Maintainer == "" &&
		f.MigrationStatus == "" && f.Search == "" && f.Depends == "" && f.Team == ""
}

// SortField identifies which field to sort sources by.
type SortField int

const (
	SortByName SortField = iota
	SortByAge
)

// SortDirection controls ascending vs descending order.
type SortDirection int

const (
	SortAsc SortDirection = iota
	SortDesc
)

// SortOrder holds the validated sort field and direction.
type SortOrder struct {
	Field     SortField
	Direction SortDirection
}

// String returns the canonical name for a SortField.
func (f SortField) String() string {
	switch f {
	case SortByAge:
		return "age"
	default:
		return "name"
	}
}

// String returns the canonical name for a SortDirection.
func (d SortDirection) String() string {
	switch d {
	case SortDesc:
		return "desc"
	default:
		return "asc"
	}
}

// ParseSortOrder extracts sort and order query parameters.
// Supported sort values: "age" (default), "name".
// Supported order values: "asc" (default), "desc".
func ParseSortOrder(r *http.Request) SortOrder {
	s := SortOrder{
		Field:     SortByAge,
		Direction: SortAsc,
	}

	switch strings.ToLower(r.URL.Query().Get("sort")) {
	case "name":
		s.Field = SortByName
	}

	switch strings.ToLower(r.URL.Query().Get("order")) {
	case "desc":
		s.Direction = SortDesc
	}

	return s
}
