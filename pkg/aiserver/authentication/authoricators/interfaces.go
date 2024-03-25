package authoricators

import (
	"context"

	"k8s.io/apiserver/pkg/authentication/authenticator"
)

type Password interface {
	AuthenticatePassword(ctx context.Context, user, password string) (*authenticator.Response, bool, error)
}
