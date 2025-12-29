package active_directory

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

func (b BasePrincipal) GetID() string   { return b.ID }
func (b BasePrincipal) GetSID() string  { return b.SID }
func (b BasePrincipal) GetName() string { return b.Name }
