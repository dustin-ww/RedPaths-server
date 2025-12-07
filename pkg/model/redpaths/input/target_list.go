package input

import "RedPaths-server/pkg/model"

type TargetListValue struct {
	CommonFields
	Value []model.Target `json:"value"`
}

func (TargetListValue) typeName() string { return "targetInput" }
