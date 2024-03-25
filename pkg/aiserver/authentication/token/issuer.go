package token

import (
	"time"

	"k8s.io/apiserver/pkg/authentication/user"
)

const (
	AccessToken  TokenType = "access_token"
	RefreshToken TokenType = "refresh_token"
	StaticToken  TokenType = "static_token"
)

type TokenType string

// Issuer issues token to user, tokens are required to perform mutating requests to resources
type Issuer interface {
	// IssueTo issues a token a User, return error if issuing process failed
	IssueTo(user user.Info, tokenType TokenType, expiresIn time.Duration) (string, error)

	// Verify verifies a token, and return a user info if it's a valid token, otherwise return error
	Verify(string) (user.Info, TokenType, error)
}
