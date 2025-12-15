package redpaths

import (
	"RedPaths-server/pkg/model"
	"fmt"
	"time"
)

type ModuleRun struct {
	ModuleKey     string         `gorm:"column:module_key" json:"module_key"`
	RunUID        string         `gorm:"column:run_uid" json:"run_uid"`
	VectorRunUID  string         `gorm:"column:vector_run_uid" json:"vector_run_uid"`
	RanAt         time.Time      `gorm:"column:ran_at" json:"ran_at"`
	ProjectUID    string         `gorm:"column:project_uid" json:"project_uid"`
	WasSuccessful bool           `gorm:"column:was_successful" json:"was_successful"`
	Targets       []model.Target `gorm:"column:targets;type:jsonb;serializer:json" json:"targets"`
	Parameters    []ModuleOption `gorm:"column:parameters;type:jsonb;serializer:json" json:"parameters"`
}

type ModuleRunBuilder struct {
	moduleRun *ModuleRun
}

func (b *ModuleRunBuilder) Build() (*ModuleRun, error) {
	if b.moduleRun.ModuleKey == "" {
		return nil, fmt.Errorf("module_key is required")
	}
	if b.moduleRun.RunUID == "" {
		return nil, fmt.Errorf("run_uid is required")
	}
	if b.moduleRun.ProjectUID == "" {
		return nil, fmt.Errorf("project_uid is required")
	}

	return b.moduleRun, nil
}

func NewModuleRunBuilder() *ModuleRunBuilder {
	return &ModuleRunBuilder{
		moduleRun: &ModuleRun{
			RanAt: time.Now(),
		},
	}
}

func (b *ModuleRunBuilder) ModuleKey(key string) *ModuleRunBuilder {
	b.moduleRun.ModuleKey = key
	return b
}

func (b *ModuleRunBuilder) RunUID(runUID string) *ModuleRunBuilder {
	b.moduleRun.RunUID = runUID
	return b
}

func (b *ModuleRunBuilder) ProjectUID(projectUID string) *ModuleRunBuilder {
	b.moduleRun.ProjectUID = projectUID
	return b
}

func (b *ModuleRunBuilder) RanAt(t time.Time) *ModuleRunBuilder {
	b.moduleRun.RanAt = t
	return b
}

func (b *ModuleRunBuilder) WasSuccessful(success bool) *ModuleRunBuilder {
	b.moduleRun.WasSuccessful = success
	return b
}

func (b *ModuleRunBuilder) Targets(targets []model.Target) *ModuleRunBuilder {
	b.moduleRun.Targets = targets
	return b
}

func (b *ModuleRunBuilder) Parameters(params []ModuleOption) *ModuleRunBuilder {
	b.moduleRun.Parameters = params
	return b
}

func (b *ModuleRunBuilder) VectorRunUID(vectorRunUID string) *ModuleRunBuilder {
	b.moduleRun.VectorRunUID = vectorRunUID
	return b
}
