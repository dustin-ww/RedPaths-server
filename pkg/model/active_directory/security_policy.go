package active_directory

import "RedPaths-server/pkg/model"

type SecurityPolicy struct {
	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// Policy attributes
	MinPwdLength     int `json:"security_policy.min_pwd_length,omitempty"`
	PwdHistoryLength int `json:"security_policy.pwd_history_length,omitempty"`
	LockoutThreshold int `json:"security_policy.lockout_threshold,omitempty"`
	LockoutDuration  int `json:"security_policy.lockout_duration,omitempty"`

	RedPathsMetadata model.RedPathsMetadata `json:"-"`
}

func (sp *SecurityPolicy) UnmarshalJSON(data []byte) error {
	type Alias SecurityPolicy
	aux := (*Alias)(sp)
	return model.UnmarshalWithMetadata(data, aux, &sp.RedPathsMetadata)
}

func (sp SecurityPolicy) MarshalJSON() ([]byte, error) {
	type Alias SecurityPolicy
	return model.MarshalWithMetadata(Alias(sp), sp.RedPathsMetadata)
}
