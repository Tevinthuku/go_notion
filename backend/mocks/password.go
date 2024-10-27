package mocks

type AnyPassword struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyPassword) Match(v interface{}) bool {
	password, ok := v.(string)
	if !ok {
		return false
	}
	return password != ""
}
