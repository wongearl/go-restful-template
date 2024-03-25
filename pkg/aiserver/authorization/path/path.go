package path

import (
	"fmt"
	"strings"

	"github.com/wongearl/go-restful-template/pkg/aiserver/authorization/authorizer"

	"k8s.io/apimachinery/pkg/util/sets"
)

// NewAuthorizer returns an authorizer which accepts a given set of paths.
// Each path is either a fully matching path or it ends in * in case a prefix match is done. A leading / is optional.
func NewAuthorizer(alwaysAllowPaths []string) (authorizer.Authorizer, error) {
	var prefixes []string
	paths := sets.NewString()
	for _, p := range alwaysAllowPaths {
		p = strings.TrimPrefix(p, "/")
		if len(p) == 0 {
			// matches "/"
			paths.Insert(p)
			continue
		}
		if strings.ContainsRune(p[:len(p)-1], '*') {
			return nil, fmt.Errorf("only trailing * allowed in %q", p)
		}
		if strings.HasSuffix(p, "*") {
			prefixes = append(prefixes, p[:len(p)-1])
		} else {
			paths.Insert(p)
		}
	}

	return authorizer.AuthorizerFunc(func(a authorizer.Attributes) (authorizer.Decision, string, error) {
		pth := strings.TrimPrefix(a.GetPath(), "/")
		if paths.Has(pth) {
			return authorizer.DecisionAllow, "", nil
		}

		for _, prefix := range prefixes {
			if strings.HasPrefix(pth, prefix) {
				return authorizer.DecisionAllow, "", nil
			}
		}

		return authorizer.DecisionNoOpinion, "", nil
	}), nil
}
