package identityprovider

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/oauth"
)

type GenericProvider interface {
	// Authenticate from remote server
	Authenticate(username string, password string) (Identity, error)
}

type GenericProviderFactory interface {
	// Type unique type of the provider
	Type() string
	// Apply the dynamic options from ai-config
	Create(options oauth.DynamicOptions) (GenericProvider, error)
}
