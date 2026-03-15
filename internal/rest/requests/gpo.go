package requests

import "RedPaths-server/pkg/model/utils/assertion"

type AddGPOLinkRequest struct {
	GPOLink          GPOLinkPayload    `json:"gpo_link" binding:"required" validate:"required"`
	GPO              GPOPayload        `json:"gpo" binding:"required" validate:"required"`
	AssertionContext assertion.Context `json:"assertion_ctx" binding:"required"`
}

type GPOLinkPayload struct {
	LinkOrder  int  `json:"link_order" binding:"required" validate:"required"`
	IsEnforced bool `json:"is_enforced"`
	IsEnabled  bool `json:"is_enabled"`
}

type GPOPayload struct {
	Name string `json:"name" binding:"required" validate:"required"`
}
