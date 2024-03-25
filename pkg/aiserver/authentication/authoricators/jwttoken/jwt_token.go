package jwttoken

import (
	"context"

	iamv1listers "github.com/wongearl/go-restful-template/pkg/client/ai/listers/iam.ai.io/v1"
	"github.com/wongearl/go-restful-template/pkg/middles/auth"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog"
)

type tokenAuthenticator struct {
	tokenOperator auth.TokenManagementInterface
	userLister    iamv1listers.UserLister
}

func NewTokenAuthenticator(tokenOperator auth.TokenManagementInterface, userLister iamv1listers.UserLister) authenticator.Token {
	return &tokenAuthenticator{
		tokenOperator: tokenOperator,
		userLister:    userLister,
	}
}

func (t *tokenAuthenticator) AuthenticateToken(ctx context.Context, token string) (*authenticator.Response, bool, error) {
	providedUser, err := t.tokenOperator.Verify(token)
	if err != nil {
		klog.Error(err)
		return nil, false, err
	}

	dbUser, err := t.userLister.Get(providedUser.GetName())
	if err != nil {
		return nil, false, err
	}

	return &authenticator.Response{
		User: &user.DefaultInfo{
			Name:   dbUser.GetName(),
			Groups: append(dbUser.Spec.Groups, user.AllAuthenticated),
		},
	}, true, nil
}
