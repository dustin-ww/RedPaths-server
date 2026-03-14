package requests

import "RedPaths-server/pkg/model/utils/assertion"

type AddDomainRequest struct {
	Name             string            `json:"name" binding:"required" validate:"required"`
	AssertionContext assertion.Context `json:"assertion_ctx" binding:"required"`
}
