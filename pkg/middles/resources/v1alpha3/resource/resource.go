package resource

import (
	"errors"

	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	"github.com/wongearl/go-restful-template/pkg/client/informers"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/customresourcedefinition"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/namespace"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/rolebinding"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/serviceaccount"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var ErrResourceNotSupported = errors.New("resource is not supported")

type ResourceGetter struct {
	clusterResourceGetters    map[schema.GroupVersionResource]v1alpha3.Interface
	namespacedResourceGetters map[schema.GroupVersionResource]v1alpha3.Interface
}

func NewResourceGetter(factory informers.InformerFactory) *ResourceGetter {
	namespacedResourceGetters := make(map[schema.GroupVersionResource]v1alpha3.Interface)
	clusterResourceGetters := make(map[schema.GroupVersionResource]v1alpha3.Interface)

	namespacedResourceGetters[schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}] = serviceaccount.New(factory.KubernetesSharedInformerFactory())
	//namespacedResourceGetters[schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}] = rolebinding.New(factory.KubernetesSharedInformerFactory())
	namespacedResourceGetters[rbacv1.SchemeGroupVersion.WithResource("rolebindings")] = rolebinding.New(factory.KubernetesSharedInformerFactory())
	clusterResourceGetters[schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}] = namespace.New(factory.KubernetesSharedInformerFactory())
	clusterResourceGetters[schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}] = customresourcedefinition.New(factory.ApiExtensionSharedInformerFactory())

	return &ResourceGetter{
		namespacedResourceGetters: namespacedResourceGetters,
		clusterResourceGetters:    clusterResourceGetters,
	}
}

// TryResource will retrieve a getter with resource name, it doesn't guarantee find resource with correct group version
// need to refactor this use schema.GroupVersionResource
func (r *ResourceGetter) TryResource(clusterScope bool, resource string) v1alpha3.Interface {
	if clusterScope {
		for k, v := range r.clusterResourceGetters {
			if k.Resource == resource {
				return v
			}
		}
	}
	for k, v := range r.namespacedResourceGetters {
		if k.Resource == resource {
			return v
		}
	}
	return nil
}

func (r *ResourceGetter) Get(resource, namespace, name string) (runtime.Object, error) {
	clusterScope := namespace == ""
	getter := r.TryResource(clusterScope, resource)
	if getter == nil {
		return nil, ErrResourceNotSupported
	}
	return getter.Get(namespace, name)
}

func (r *ResourceGetter) List(resource, namespace string, query *query.Query) (*api.ListResult, error) {
	clusterScope := namespace == ""
	getter := r.TryResource(clusterScope, resource)
	if getter == nil {
		return nil, ErrResourceNotSupported
	}
	return getter.List(namespace, query)
}
