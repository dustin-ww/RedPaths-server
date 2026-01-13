package active_directory

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/utils"
	"time"
)

type User struct {

	// Internal & Interface
	core.BasePrincipal

	// Specific
	SAMAccountName    string          `json:"user.sam_account_name,omitempty"`
	UPN               string          `json:"user.upn,omitempty"`
	IsDisabled        bool            `json:"user.is_disabled,omitempty"`
	IsLocked          bool            `json:"user.is_locked,omitempty"`
	IsServiceAccount  bool            `json:"user.is_service_account,omitempty"`
	LastLogon         time.Time       `json:"user.last_login,omitempty"`
	PwdLastSet        time.Time       `json:"user.pwd_last_set,omitempty"`
	BadPwdCount       int             `json:"user.bad_pwd_count,omitempty"`
	AllowedToDelegate bool            `json:"user.allowed_to_delegate,omitempty"`
	HasSPN            bool            `json:"user.has_spn,omitempty"`
	HasSession        []*utils.UIDRef `json:"user.has_session,omitempty"`
	MemberOf          []*utils.UIDRef `json:"group.member_of,omitempty"`

	Kerberoastable bool `json:"user.kerberoastable,omitempty"`
	ASREPRoastable bool `json:"user.asrep_roastable,omitempty"`

	// Risk
	RiskScore   int      `json:"risk_score,omitempty"`
	RiskReasons []string `json:"risk_reason,omitempty"`

	//// Credentials
	//Password       string `json:"password,omitempty"`
	//NTLMHash       string `json:"ntlm_hash,omitempty"`
	//CredentialType string `json:"credential_type,omitempty"`

	// Privileges
	IsLocalAdmin  bool `json:"user.is_local_admin,omitempty"`
	IsDomainAdmin bool `json:"user.is_domain_admin,omitempty"`

	// Delegation
	/*	TrustedForDelegation    bool `json:"trusted_for_delegation,omitempty"`
		UnconstrainedDelegation bool `json:"unconstrained_delegation,omitempty"`
	*/
	// Usage
	//Workstations []string `json:"workstations,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (u *User) PrincipalType() PrincipalType {
	return PrincipalUser
}

func (u *User) UnmarshalJSON(data []byte) error {
	type Alias User
	aux := (*Alias)(u)
	return core.UnmarshalWithMetadata(data, aux, &u.RedPathsMetadata)
}

func (u User) MarshalJSON() ([]byte, error) {
	type Alias User
	return core.MarshalWithMetadata(Alias(u), u.RedPathsMetadata)
}
