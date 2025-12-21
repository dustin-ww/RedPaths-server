package model

import (
	"RedPaths-server/pkg/model/utils"
	"time"
)

type ADUser struct {
	UID  string
	Name string

	// Identity
	SAMAccountName string
	UPN            string
	SID            string
	AccountType    string

	// Credentials
	Password       string
	NTLMHash       string
	CredentialType string

	// Privileges
	IsAdmin       bool
	IsDomainAdmin bool
	MemberOf      []*utils.UIDRef

	// Kerberos
	SPNs           []string
	Kerberoastable bool
	ASREPRoastable bool

	// Delegation
	TrustedForDelegation    bool
	UnconstrainedDelegation bool

	// Usage
	LastLogon    time.Time
	Workstations []string

	// Risk
	RiskScore   int
	RiskReasons []string

	// History related
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	LastSeenBy   string    `json:"last_seen_by,omitempty"`
}
