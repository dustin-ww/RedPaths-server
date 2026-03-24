package upsert

import "RedPaths-server/pkg/model/utils/assertion"

type Input[T any] struct {
	Entity       T
	ProjectUID   string
	ParentUID    *string // nil → orphaned
	ParentType   string
	AssertionCtx assertion.Context
	Actor        string
}

func (i Input[T]) Resolved() (subjectUID, subjectType string, hasParent bool) {
	if i.ParentUID != nil {
		return *i.ParentUID, i.ParentType, true
	}
	return i.ProjectUID, "Project", false
}
