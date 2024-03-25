package basictoken

import (
	"errors"
	"net/http"

	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/authoricators"

	"k8s.io/apiserver/pkg/authentication/authenticator"
)

type Authenticator struct {
	auth authoricators.Password
}

func New(auth authoricators.Password) *Authenticator {
	return &Authenticator{auth}
}

var invalidToken = errors.New("invalid basic token")

func (a *Authenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {

	username, password, ok := req.BasicAuth()

	if !ok {
		return nil, false, nil
	}

	resp, ok, err := a.auth.AuthenticatePassword(req.Context(), username, password)

	// If the token authenticator didn't error, provide a default error
	if !ok && err == nil {
		err = invalidToken
	}

	return resp, ok, err
}
