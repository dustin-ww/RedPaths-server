package redpaths

import (
	"time"
)

type VectorRun struct {
	RunUID     string    `gorm:"column:run_uid" json:"run_uid"`
	RanAt      time.Time `gorm:"column:ran_at" json:"ran_at"`
	ProjectUID string    `gorm:"column:project_uid" json:"project_uid"`
	// TODO: Optimize (flat graph)
	Graph *InheritanceGraph `gorm:"column:graph;type:jsonb;serializer:json" json:"graph"`
}

type VectorRunBuilder struct {
	runUID     string
	ranAt      time.Time
	projectUID string
	graph      *InheritanceGraph
}

func NewVectorRunBuilder() *VectorRunBuilder {
	return &VectorRunBuilder{
		ranAt: time.Now(),
	}
}

func (b *VectorRunBuilder) WithRunUID(uid string) *VectorRunBuilder {
	b.runUID = uid
	return b
}

func (b *VectorRunBuilder) WithRanAt(t time.Time) *VectorRunBuilder {
	b.ranAt = t
	return b
}

func (b *VectorRunBuilder) WithProjectUID(uid string) *VectorRunBuilder {
	b.projectUID = uid
	return b
}

func (b *VectorRunBuilder) WithGraph(g *InheritanceGraph) *VectorRunBuilder {
	b.graph = g
	return b
}

func (b *VectorRunBuilder) Build() *VectorRun {
	return &VectorRun{
		RunUID:     b.runUID,
		RanAt:      b.ranAt,
		ProjectUID: b.projectUID,
		Graph:      b.graph,
	}
}
