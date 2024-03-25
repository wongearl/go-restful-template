package auth

import (
	"fmt"
	"time"

	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/oauth"
	authoptions "github.com/wongearl/go-restful-template/pkg/aiserver/authentication/options"
	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/token"
	"github.com/wongearl/go-restful-template/pkg/client/cache"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog"
)

type TokenManagementInterface interface {
	// Verify verifies a token, and return a User if it's a valid token, otherwise return error
	Verify(token string) (user.Info, error)
	// IssueTo issues a token a User, return error if issuing process failed
	IssueTo(user user.Info, accessTokenMaxAge, accessTokenInactivityTimeout *time.Duration) (*oauth.Token, error)
}

type tokenOperator struct {
	issuer  token.Issuer
	options *authoptions.AuthenticationOptions
	cache   cache.Interface
}

func NewTokenOperator(cache cache.Interface, options *authoptions.AuthenticationOptions) TokenManagementInterface {
	operator := &tokenOperator{
		issuer:  token.NewTokenIssuer(options.JwtSecret, options.MaximumClockSkew),
		options: options,
		cache:   cache,
	}
	return operator
}

func (t tokenOperator) Verify(tokenStr string) (user.Info, error) {
	authenticated, tokenType, err := t.issuer.Verify(tokenStr)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	if t.options.OAuthOptions.AccessTokenMaxAge == 0 ||
		tokenType == token.StaticToken {
		return authenticated, nil
	}
	if err := t.tokenCacheValidate(authenticated.GetName(), tokenStr); err != nil {
		klog.Error(err)
		return nil, err
	}
	return authenticated, nil
}

func (t tokenOperator) IssueTo(user user.Info, accessTokenMaxAge, accessTokenInactivityTimeout *time.Duration) (*oauth.Token, error) {
	accessTokenExpiresIn := t.options.OAuthOptions.AccessTokenMaxAge
	refreshTokenExpiresIn := accessTokenExpiresIn + t.options.OAuthOptions.AccessTokenInactivityTimeout
	tokenType := "Bearer"
	createTokenType := token.AccessToken
	// use external configuration
	if accessTokenMaxAge != nil && accessTokenInactivityTimeout != nil {
		accessTokenExpiresIn = *accessTokenMaxAge
		refreshTokenExpiresIn = accessTokenExpiresIn + *accessTokenInactivityTimeout
		tokenType = "static_token"
		createTokenType = token.StaticToken
	}

	accessToken, err := t.issuer.IssueTo(user, createTokenType, accessTokenExpiresIn)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	refreshToken, err := t.issuer.IssueTo(user, token.RefreshToken, refreshTokenExpiresIn)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	result := &oauth.Token{
		AccessToken:  accessToken,
		TokenType:    tokenType,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenExpiresIn.Seconds()),
	}

	if accessTokenExpiresIn > 0 {
		if err = t.cacheToken(user.GetName(), accessToken, accessTokenExpiresIn); err != nil {
			klog.Error(err)
			return nil, err
		}
		if err = t.cacheToken(user.GetName(), refreshToken, refreshTokenExpiresIn); err != nil {
			klog.Error(err)
			return nil, err
		}
	}

	return result, nil
}

func (t tokenOperator) tokenCacheValidate(username, token string) error {
	key := fmt.Sprintf("ai:user:%s:token:%s", username, token)
	if exist, err := t.cache.Exists(key); err != nil {
		return err
	} else if !exist {
		return fmt.Errorf("token not found in cache")
	}
	return nil
}

func (t tokenOperator) cacheToken(username, token string, duration time.Duration) error {
	key := fmt.Sprintf("ai:user:%s:token:%s", username, token)
	if err := t.cache.Set(key, token, duration); err != nil {
		klog.Error(err)
		return err
	}
	return nil
}
