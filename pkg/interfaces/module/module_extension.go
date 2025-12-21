package module

type PrerequisiteType string

const (
	PrereqCredentials   PrerequisiteType = "credentials"
	PrereqNetworkAccess PrerequisiteType = "network_access"
	PrereqPrivilege     PrerequisiteType = "privilege"
	PrereqKnowledge     PrerequisiteType = "knowledge"
	PrereqTool          PrerequisiteType = "tool"
	PrereqModule        PrerequisiteType = "module"
)

type Prerequisite struct {
	Type        PrerequisiteType
	Name        string
	Description string
	Required    bool
	Conditions  string
}

type Capability struct {
	Type        string
	Name        string
	Description string
	Confidence  float64
	Metadata    map[string]interface{}
}

type ExecutionContext struct {
	AcquiredCapabilities map[string]*Capability
	ExecutedModules      map[string]bool
	FailedModules        map[string]error
	TargetEnvironment    map[string]interface{}
	CurrentPrivileges    []string
}
