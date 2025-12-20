package history

type Diffable interface {
	EntityUID() string
	EntityType() string
	Diff(other any) []FieldChange
}
