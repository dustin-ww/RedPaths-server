package active_directory

type SecurityPrincipal interface {
	GetUID() string
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
