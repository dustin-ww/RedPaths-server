package model

import (
	"RedPaths-server/pkg/model/utils"
	"time"
)

type ADUser struct {
	UID  string `json:"uid,omitempty"`
	Name string `json:"name,omitempty"`

	// Identity
	SAMAccountName string `json:"sam_account_name,omitempty"`
	UPN            string `json:"upn,omitempty"`
	SID            string `json:"sid,omitempty"`
	AccountType    string `json:"account_type,omitempty"`

	// Credentials
	Password       string `json:"password,omitempty"`
	NTLMHash       string `json:"ntlm_hash,omitempty"`
	CredentialType string `json:"credential_type,omitempty"`

	// Privileges
	IsAdmin       bool            `json:"is_admin,omitempty"`
	IsDomainAdmin bool            `json:"is_domain_admin,omitempty"`
	MemberOf      []*utils.UIDRef `json:"member_of,omitempty"`

	// Kerberos
	SPNs           []string `json:"spns,omitempty"`
	Kerberoastable bool     `json:"kerberoastable,omitempty"`
	ASREPRoastable bool     `json:"asrep_roastable,omitempty"`

	// Delegation
	TrustedForDelegation    bool `json:"trusted_for_delegation,omitempty"`
	UnconstrainedDelegation bool `json:"unconstrained_delegation,omitempty"`

	// Usage
	LastLogon    time.Time `json:"last_logon,omitempty"`
	Workstations []string  `json:"workstations,omitempty"`

	// Risk
	RiskScore   int      `json:"risk_score,omitempty"`
	RiskReasons []string `json:"risk_reason,omitempty"`

	// History related
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	LastSeenBy   string    `json:"last_seen_by,omitempty"`
}
