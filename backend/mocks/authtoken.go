package mocks

// TokenGeneratorMock implements TokenGenerator interface for testing purposes
type TokenGeneratorMock struct{}

func (t *TokenGeneratorMock) GenerateToken(userID int64) (string, error) {
	return "token", nil
}
