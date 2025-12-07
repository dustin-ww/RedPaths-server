package query

import "time"

type LogQueryOptions struct {
	// Pagination
	Page     int
	PageSize int

	// Search/Filter
	SearchTerm string
	EventTypes []string
	ModuleKeys []string

	// Sorting
	SortBy    string
	SortOrder string

	// Time range
	StartTime *time.Time
	EndTime   *time.Time
}
