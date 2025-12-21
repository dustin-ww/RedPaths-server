package model

import (
	"RedPaths-server/pkg/model/utils"
	"time"
)

type ADGroup struct {
	UID  string `json:"uid,omitempty"`
	Name string `json:"name,omitempty"`
	SID  string `json:"sid,omitempty"`

	// Scope & Type
	GroupScope string `json:"group_scope,omitempty"`
	// domain_local | global | universal

	GroupType string `json:"group_type,omitempty"`
	// security | distribution

	// Membership
	Members  []*utils.UIDRef `json:"members,omitempty"`
	MemberOf []*utils.UIDRef `json:"member_of,omitempty"`

	// Privileges
	Privileges []string `json:"privileges,omitempty"`

	// AD relevance
	IsPrivileged bool `json:"is_privileged,omitempty"`
	IsBuiltIn    bool `json:"is_built_in,omitempty"`

	// Exploitation flags
	CanDCSync       bool `json:"can_dcsync,omitempty"`
	CanLogonLocally bool `json:"can_logon_locally,omitempty"`
	CanRDP          bool `json:"can_rdp,omitempty"`

	// Risk
	RiskScore   int      `json:"risk_score,omitempty"`
	RiskReasons []string `json:"risk_reasons,omitempty"`

	// History
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	LastSeenBy   string    `json:"last_seen_by,omitempty"`
}
