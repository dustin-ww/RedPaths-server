package requests

import "RedPaths-server/pkg/model/utils/assertion"

type AddHostRequest struct {
	Ip               string            `json:"ipAddress" binding:"required"`
	AssertionContext assertion.Context `json:"assertion_ctx" binding:"required"`
}
