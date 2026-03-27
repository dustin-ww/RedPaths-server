package requests

import "RedPaths-server/pkg/model/utils/assertion"

type CreateActiveDirectory struct {
	ForestName            string             `json:"forest_name" binding:"required"`
	ForestFunctionalLevel string             `json:"forest_functional_level,omitempty"`
	AssertionContext      *assertion.Context `json:"assertion_ctx"`
}
