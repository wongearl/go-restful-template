package union

import (
	"strings"

	"github.com/wongearl/go-restful-template/pkg/aiserver/authorization/authorizer"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/authentication/user"
)

// Following code copied from k8s.io/apiserver/pkg/authorization/union to avoid import collision

// unionAuthzHandler authorizer against a chain of authorizer.Authorizer
type unionAuthzHandler []authorizer.Authorizer

// New returns an authorizer that authorizes against a chain of authorizer.Authorizer objects
func New(authorizationHandlers ...authorizer.Authorizer) authorizer.Authorizer {
	return unionAuthzHandler(authorizationHandlers)
}

// Authorizes against a chain of authorizer.Authorizer objects and returns nil if successful and returns error if unsuccessful
func (authzHandler unionAuthzHandler) Authorize(a authorizer.Attributes) (authorizer.Decision, string, error) {
	var (
		errlist    []error
		reasonlist []string
	)

	for _, currAuthzHandler := range authzHandler {
		decision, reason, err := currAuthzHandler.Authorize(a)

		if err != nil {
			errlist = append(errlist, err)
		}
		if len(reason) != 0 {
			reasonlist = append(reasonlist, reason)
		}
		switch decision {
		case authorizer.DecisionAllow, authorizer.DecisionDeny:
			return decision, reason, err
		case authorizer.DecisionNoOpinion:
			// continue to the next authorizer
		}
	}

	return authorizer.DecisionNoOpinion, strings.Join(reasonlist, "\n"), utilerrors.NewAggregate(errlist)
}

// unionAuthzRulesHandler authorizer against a chain of authorizer.RuleResolver
type unionAuthzRulesHandler []authorizer.RuleResolver

// NewRuleResolvers returns an authorizer that authorizes against a chain of authorizer.Authorizer objects
func NewRuleResolvers(authorizationHandlers ...authorizer.RuleResolver) authorizer.RuleResolver {
	return unionAuthzRulesHandler(authorizationHandlers)
}

// RulesFor against a chain of authorizer.RuleResolver objects and returns nil if successful and returns error if unsuccessful
func (authzHandler unionAuthzRulesHandler) RulesFor(user user.Info, namespace string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	var (
		errList              []error
		resourceRulesList    []authorizer.ResourceRuleInfo
		nonResourceRulesList []authorizer.NonResourceRuleInfo
	)
	incompleteStatus := false

	for _, currAuthzHandler := range authzHandler {
		resourceRules, nonResourceRules, incomplete, err := currAuthzHandler.RulesFor(user, namespace)

		if incomplete == true {
			incompleteStatus = true
		}
		if err != nil {
			errList = append(errList, err)
		}
		if len(resourceRules) > 0 {
			resourceRulesList = append(resourceRulesList, resourceRules...)
		}
		if len(nonResourceRules) > 0 {
			nonResourceRulesList = append(nonResourceRulesList, nonResourceRules...)
		}
	}

	return resourceRulesList, nonResourceRulesList, incompleteStatus, utilerrors.NewAggregate(errList)
}
