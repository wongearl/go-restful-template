package identityprovider

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/oauth"
)

type OAuthProvider interface {
	// IdentityExchange exchange identity from remote server
	IdentityExchange(code string) (Identity, error)
}

type OAuthProviderFactory interface {
	// Type unique type of the provider
	Type() string
	// Apply the dynamic options from ai-config
	Create(options oauth.DynamicOptions) (OAuthProvider, error)
}
