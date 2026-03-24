package requests

import (
	"RedPaths-server/pkg/model/engine"
	"RedPaths-server/pkg/model/utils/assertion"
)

type AddCapabilityRequest struct {
	Name             string            `json:"name" binding:"required" validate:"required"`
	Scope            string            `json:"scope" binding:"required" validate:"required"`
	AssertionContext assertion.Context `json:"assertion_ctx" binding:"required"`
	SourceType       engine.SourceType `json:"source_type"`
	Precondition     string            `json:"precondition"`
	RiskLevel        int               `json:"risk_level" binding:"required" validate:"required"`
}
