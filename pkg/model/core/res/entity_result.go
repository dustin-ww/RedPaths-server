package res

import (
	"RedPaths-server/pkg/model/active_directory/gpo"
	"RedPaths-server/pkg/model/active_directory/priv"
	"RedPaths-server/pkg/model/core"
	"time"
)

type EntityResult[T any] struct {
	Entity     T                 `json:"entity"`
	Assertions []*core.Assertion `json:"assertions"`
	ACL        *priv.ACL         `json:"acl,omitempty"`
	Metadata   *ResultMetadata   `json:"metadata,omitempty"`
}

type GPOResult[T any] struct {
	GPOLink           *gpo.Link         `json:"link"`
	GPOLinkAssertions []*core.Assertion `json:"gpo_link_assertions"`
	GPO               *gpo.GPO          `json:"gpo"`
	GPOAssertions     []*core.Assertion `json:"gpo_assertions"`
	ACL               *priv.ACL         `json:"acl,omitempty"`
	Metadata          *ResultMetadata   `json:"metadata,omitempty"`
}

type GPOLinkEntry struct {
	GPOLink           *gpo.Link         `json:"link"`
	GPOLinkAssertions []*core.Assertion `json:"assertions"`
	GPO               *gpo.GPO          `json:"gpo"`
	GPOAssertions     []*core.Assertion `json:"gpo_assertions"`
}

type GPOQueryResult struct {
	Entries  []*GPOLinkEntry `json:"entries"`
	Metadata *ResultMetadata `json:"metadata,omitempty"`
}

type ResultMetadata struct {
	Source         string    `json:"source"`
	ScanTimestamp  time.Time `json:"scan_timestamp"`
	EntityCount    int       `json:"entity_count"`
	AssertionCount int       `json:"assertion_count"`
}
