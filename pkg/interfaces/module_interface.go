package interfaces

import (
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/model/rpsdk"
	"RedPaths-server/pkg/sse"
)

type RedPathsModule interface {
	ConfigKey() string
	ExecuteModule(params *input.Parameter, logger *sse.SSELogger) error
	SetServices(services *rpsdk.Services)
	GetMetadata() *ModuleMetadata
}

type PrerequisiteType string

const (
	PrereqCredentials   PrerequisiteType = "credentials"
	PrereqNetworkAccess PrerequisiteType = "network_access"
	PrereqPrivilege     PrerequisiteType = "privilege"
	PrereqKnowledge     PrerequisiteType = "knowledge"
	PrereqTool          PrerequisiteType = "tool"
	PrereqModule        PrerequisiteType = "module" // Abhängigkeit von anderem Modul
)

// Prerequisite beschreibt eine Voraussetzung für ein Modul
type Prerequisite struct {
	Type        PrerequisiteType
	Name        string
	Description string
	Required    bool   // Muss erfüllt sein oder nur empfohlen
	Conditions  string // Muss erfüllt sein oder nur empfohlen

}

// Capability beschreibt was ein Modul bereitstellt/erreicht
type Capability struct {
	Type        string // z.B. "credentials", "privilege_escalation", "lateral_movement"
	Name        string
	Description string
	Confidence  float64 // 0.0 - 1.0: Wahrscheinlichkeit dass es funktioniert
	Metadata    map[string]interface{}
}

// ExecutionContext enthält den aktuellen Zustand der Ausführung
type ExecutionContext struct {
	AcquiredCapabilities map[string]*Capability
	ExecutedModules      map[string]bool
	FailedModules        map[string]error
	TargetEnvironment    map[string]interface{}
	CurrentPrivileges    []string
}

// ModuleMetadata erweitert das bestehende Plugin-Interface
type ModuleMetadata struct {
	Name          string
	Category      string // z.B. "reconnaissance", "exploitation", "post-exploitation"
	Description   string
	Prerequisites []*Prerequisite
	Provides      []*Capability
	Risk          int // 1-10: Wie riskant ist die Ausführung
	Stealth       int // 1-10: Wie auffällig ist das Modul
	Complexity    int // 1-10: Wie komplex ist die Ausführung
}
