package error

import "errors"

// ErrNotFound error when entity is not existing
var ErrNotFound = errors.New("entity not found")
