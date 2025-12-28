package error

import "errors"

// ErrNotFound error when entity is not existing
var ErrNotFound = errors.New("entity not found")

var ErrProjectUpdate = errors.New("project update failed")
var ErrDomainUpdate = errors.New("domain update failed")
