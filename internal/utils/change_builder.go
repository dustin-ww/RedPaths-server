package utils

import (
	"RedPaths-server/pkg/model/redpaths/history"
	"time"

	"github.com/google/uuid"
)

func BuildChange(
	before history.Diffable,
	after history.Diffable,
	opts ...func(*history.Change),
) *history.Change {

	changes := before.Diff(after)
	if len(changes) == 0 {
		return nil
	}

	c := &history.Change{
		UID:        uuid.New(),
		EntityType: before.EntityType(),
		EntityUID:  before.EntityUID(),
		Changes:    changes,
		ChangedAt:  time.Now().UTC(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func WithActor(actor string) func(actor *history.Change) {
	return func(c *history.Change) {
		c.ChangedBy = actor
	}
}

func WithReason(reason string) func(*history.Change) {
	return func(c *history.Change) {
		c.ChangeReason = reason
	}
}
