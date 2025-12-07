package redpaths

type InheritanceGraph struct {
	Nodes []*Module           `json:"nodes"`
	Edges []*ModuleDependency `json:"edges"`
}
