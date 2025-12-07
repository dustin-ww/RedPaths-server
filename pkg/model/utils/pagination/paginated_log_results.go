package pagination

import "RedPaths-server/pkg/model/redpaths"

type PaginatedLogResult struct {
	Logs       []*redpaths.LogEntry
	TotalCount int64
	Page       int
	PageSize   int
	TotalPages int
}
