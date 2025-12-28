package model

type SecurityPrincipal interface {
	GetID() string
	GetSID() string
	GetName() string
	PrincipalType() PrincipalType
}

type PrincipalType string

const (
	PrincipalUser           PrincipalType = "User"
	PrincipalGroup          PrincipalType = "Group"
	PrincipalComputer       PrincipalType = "Computer"
	PrincipalServiceAccount PrincipalType = "ServiceAccount"
)

type BasePrincipal struct {
	ID          string
	Name        string
	SID         string
	Description string
	DomainID    string
}

func (b BasePrincipal) GetID() string   { return b.ID }
func (b BasePrincipal) GetSID() string  { return b.SID }
func (b BasePrincipal) GetName() string { return b.Name }
