package mocks

type TokenGeneratorMock struct{}

func (t *TokenGeneratorMock) GenerateToken(userID int64) (string, error) {
	return "token", nil
}
