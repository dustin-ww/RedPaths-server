package gpo

import (
	"RedPaths-server/pkg/model/core"
)

type Setting struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	//Specific
	SettingType int    `json:"gpo_setting.setting_type,omitempty"`
	Value       string `json:"gpo_setting.value,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (gs *Setting) UnmarshalJSON(data []byte) error {
	type Alias Setting
	aux := (*Alias)(gs)
	return core.UnmarshalWithMetadata(data, aux, &gs.RedPathsMetadata)
}

func (gs Setting) MarshalJSON() ([]byte, error) {
	type Alias Setting
	return core.MarshalWithMetadata(Alias(gs), gs.RedPathsMetadata)
}
