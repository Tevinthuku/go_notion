package mocks

import (
	"github.com/gofrs/uuid/v5"
)

type AnyUUID struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyUUID) Match(v interface{}) bool {
	_, ok := v.(uuid.UUID)
	return ok
}
