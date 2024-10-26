package mocks

type AnyPassword struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyPassword) Match(v interface{}) bool {
	_, ok := v.(string)
	return ok
}
