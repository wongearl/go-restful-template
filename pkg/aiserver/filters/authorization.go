package filters

import (
	"context"
	"errors"
	"net/http"

	"github.com/wongearl/go-restful-template/pkg/aiserver/authorization/authorizer"
	"github.com/wongearl/go-restful-template/pkg/aiserver/request"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/klog"
)

// WithAuthorization passes all authorized requests on to handler, and returns forbidden error otherwise.
func WithAuthorization(handler http.Handler, authorizers authorizer.Authorizer) http.Handler {
	if authorizers == nil {
		klog.Warningf("Authorization is disabled")
		return handler
	}

	defaultSerializer := serializer.NewCodecFactory(runtime.NewScheme()).WithoutConversion()

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		attributes, err := getAuthorizerAttributes(ctx)
		if err != nil {
			responsewriters.InternalError(w, req, err)
		}

		authorized, reason, err := authorizers.Authorize(attributes)
		if authorized == authorizer.DecisionAllow {
			handler.ServeHTTP(w, req)
			return
		}

		if err != nil {
			responsewriters.InternalError(w, req, err)
			return
		}

		klog.V(4).Infof("Forbidden: %#v, Reason: %q", req.RequestURI, reason)
		responsewriters.Forbidden(ctx, attributes, w, req, reason, defaultSerializer)
	})
}

func getAuthorizerAttributes(ctx context.Context) (authorizer.Attributes, error) {
	attribs := authorizer.AttributesRecord{}

	user, ok := request.UserFrom(ctx)
	if ok {
		attribs.User = user
	}

	requestInfo, found := request.RequestInfoFrom(ctx)
	if !found {
		return nil, errors.New("no RequestInfo found in the context")
	}

	// Start with common attributes that apply to resource and non-resource requests
	attribs.ResourceScope = requestInfo.ResourceScope
	attribs.ResourceRequest = requestInfo.IsResourceRequest
	attribs.Path = requestInfo.Path
	attribs.Verb = requestInfo.Verb
	attribs.Cluster = requestInfo.Cluster
	attribs.Workspace = requestInfo.Workspace
	attribs.KubernetesRequest = requestInfo.IsKubernetesRequest
	attribs.APIGroup = requestInfo.APIGroup
	attribs.APIVersion = requestInfo.APIVersion
	attribs.Resource = requestInfo.Resource
	attribs.Subresource = requestInfo.Subresource
	attribs.Namespace = requestInfo.Namespace
	attribs.DevOps = requestInfo.DevOps
	attribs.Name = requestInfo.Name

	return &attribs, nil
}
