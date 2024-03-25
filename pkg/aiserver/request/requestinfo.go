package request

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/wongearl/go-restful-template/pkg/api"
	"github.com/wongearl/go-restful-template/pkg/constants"
	"github.com/wongearl/go-restful-template/pkg/utils/iputil"

	"k8s.io/apimachinery/pkg/api/validation/path"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	k8srequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog"
)

type RequestInfoResolver interface {
	NewRequestInfo(req *http.Request) (*RequestInfo, error)
}

func (r *RequestInfoFactory) NewRequestInfo(req *http.Request) (*RequestInfo, error) {

	requestInfo := RequestInfo{
		IsKubernetesRequest: false,
		RequestInfo: &k8srequest.RequestInfo{
			Path: req.URL.Path,
			Verb: req.Method,
		},
		Workspace: api.WorkspaceNone,
		Cluster:   api.ClusterNone,
		SourceIP:  iputil.RemoteIp(req),
		UserAgent: req.UserAgent(),
	}

	defer func() {
		prefix := requestInfo.APIPrefix
		if prefix == "" {
			currentParts := splitPath(requestInfo.Path)
			//Proxy discovery API
			if len(currentParts) > 0 && len(currentParts) < 3 {
				prefix = currentParts[0]
			}
		}
		if kubernetesAPIPrefixes.Has(prefix) {
			requestInfo.IsKubernetesRequest = true
		}
	}()

	currentParts := splitPath(req.URL.Path)

	if len(currentParts) < 3 {
		return &requestInfo, nil
	}

	if !r.APIPrefixes.Has(currentParts[0]) {
		// return a non-resource request
		return &requestInfo, nil

	}

	requestInfo.APIPrefix = currentParts[0]

	currentParts = currentParts[1:]

	if !r.GrouplessAPIPrefixes.Has(requestInfo.APIPrefix) {
		// one part (APIPrefix) has already been consumed, so this is actually "do we have four parts?"
		if len(currentParts) < 3 {
			// return a non-resource request
			return &requestInfo, nil
		}
		requestInfo.APIGroup = currentParts[0]
		currentParts = currentParts[1:]
	}
	requestInfo.IsResourceRequest = true
	requestInfo.APIVersion = currentParts[0]
	currentParts = currentParts[1:]
	if len(currentParts) > 0 && specialVerbs.Has(currentParts[0]) {
		if len(currentParts) < 2 {
			return &requestInfo, fmt.Errorf("unable to determine kind and namespace from url: %v", req.URL)
		}
		requestInfo.Verb = currentParts[0]
		currentParts = currentParts[1:]
	} else {
		switch req.Method {
		case "POST":
			requestInfo.Verb = "create"
		case "GET", "HEAD":
			requestInfo.Verb = "get"
		case "PUT":
			requestInfo.Verb = "update"
		case "PATCH":
			requestInfo.Verb = "patch"
		case "DELETE":
			requestInfo.Verb = "delete"
		default:
			requestInfo.Verb = ""
		}
	}

	// URL forms: /workspaces/{workspace}/*

	if currentParts[0] == "workspaces" {
		if len(currentParts) > 1 {
			requestInfo.Workspace = currentParts[1]
		}
		if len(currentParts) > 2 {
			currentParts = currentParts[2:]
		}
	}

	// URL forms: /namespaces/{namespace}/{kind}/*, where parts are adjusted to be relative to kind
	if currentParts[0] == "namespaces" {
		if len(currentParts) > 1 {
			requestInfo.Namespace = currentParts[1]
			// if there is another step after the namespace name and it is not a known namespace subresource
			// move currentParts to include it as a resource in its own right
			if len(currentParts) > 2 && !namespaceSubresources.Has(currentParts[2]) {
				currentParts = currentParts[2:]
			}
		}
	} else {
		requestInfo.Namespace = metav1.NamespaceNone
		requestInfo.DevOps = metav1.NamespaceNone
	}

	// parsing successful, so we now know the proper value for .Parts
	requestInfo.Parts = currentParts
	// parts look like: resource/resourceName/subresource/other/stuff/we/don't/interpret
	switch {
	case len(requestInfo.Parts) >= 3 && !specialVerbsNoSubresources.Has(requestInfo.Verb):
		requestInfo.Subresource = requestInfo.Parts[2]
		fallthrough
	case len(requestInfo.Parts) >= 2:
		requestInfo.Name = requestInfo.Parts[1]
		fallthrough
	case len(requestInfo.Parts) >= 1:
		requestInfo.Resource = requestInfo.Parts[0]
	}
	requestInfo.ResourceScope = r.resolveResourceScope(requestInfo)

	// if there's no name on the request and we thought it was a get before, then the actual verb is a list or a watch
	if len(requestInfo.Name) == 0 && requestInfo.Verb == "get" {
		opts := metainternalversion.ListOptions{}
		if err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metav1.SchemeGroupVersion, &opts); err != nil {
			// An error in parsing request will result in default to "list" and not setting "name" field.
			klog.Errorf("Couldn't parse request %#v: %v", req.URL.Query(), err)
			// Reset opts to not rely on partial results from parsing.
			// However, if watch is set, let's report it.
			opts = metainternalversion.ListOptions{}
			if values := req.URL.Query()["watch"]; len(values) > 0 {
				switch strings.ToLower(values[0]) {
				case "false", "0":
				default:
					opts.Watch = true
				}
			}
		}

		if opts.Watch {
			requestInfo.Verb = "watch"
		} else {
			requestInfo.Verb = "list"
		}

		if opts.FieldSelector != nil {
			if name, ok := opts.FieldSelector.RequiresExactMatch("metadata.name"); ok {
				if len(path.IsValidPathSegmentName(name)) == 0 {
					requestInfo.Name = name
				}
			}
		}
	}

	// URL forms: /api/v1/watch/namespaces?labelSelector=ai.io/workspace=system-workspace

	if requestInfo.Verb == "watch" {
		selector := req.URL.Query().Get("labelSelector")
		if strings.HasPrefix(selector, workspaceSelectorPrefix) {
			workspace := strings.TrimPrefix(selector, workspaceSelectorPrefix)
			requestInfo.Workspace = workspace
			requestInfo.ResourceScope = WorkspaceScope
		}
	}

	// if there's no name on the request and we thought it was a delete before, then the actual verb is deletecollection
	if len(requestInfo.Name) == 0 && requestInfo.Verb == "delete" {
		requestInfo.Verb = "deletecollection"
	}
	return &requestInfo, nil

}

type requestInfoKeyType int

const requestInfoKey requestInfoKeyType = iota

var kubernetesAPIPrefixes = sets.NewString("api", "apis")

var namespaceSubresources = sets.NewString("status", "finalize")

var specialVerbs = sets.NewString("proxy", "watch")

var specialVerbsNoSubresources = sets.NewString("proxy")

type RequestInfo struct {
	*k8srequest.RequestInfo

	// IsKubernetesRequest indicates whether or not the request should be handled by kubernetes
	IsKubernetesRequest bool

	// Workspace of requested resource, for non-workspaced resources, this may be empty
	Workspace string

	// Cluster of requested resource, this is empty in single-cluster environment
	Cluster string

	// DevOps project of requested resource
	DevOps string

	// Scope of requested resource.
	ResourceScope string

	// Source IP
	SourceIP string

	// User agent
	UserAgent string
}

type RequestInfoFactory struct {
	APIPrefixes          sets.String
	GrouplessAPIPrefixes sets.String
	GlobalResources      []schema.GroupResource
}

const (
	GlobalScope             = "Global"
	WorkspaceScope          = "Workspace"
	NamespaceScope          = "Namespace"
	workspaceSelectorPrefix = constants.WorkspaceLabelKey + "="
)

func (r *RequestInfoFactory) resolveResourceScope(request RequestInfo) string {

	if request.Namespace != "" {
		return NamespaceScope
	}

	if request.Workspace != "" {
		return WorkspaceScope
	}

	return GlobalScope
}

func RequestInfoFrom(ctx context.Context) (*RequestInfo, bool) {
	info, ok := ctx.Value(requestInfoKey).(*RequestInfo)
	return info, ok
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

func WithRequestInfo(parent context.Context, info *RequestInfo) context.Context {
	return k8srequest.WithValue(parent, requestInfoKey, info)
}
