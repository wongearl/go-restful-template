package clusterrole

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
)

type clusterrolesGetter struct {
	sharedInformers informers.SharedInformerFactory
}

func New(sharedInformers informers.SharedInformerFactory) v1alpha3.Interface {
	return &clusterrolesGetter{sharedInformers: sharedInformers}
}

func (d *clusterrolesGetter) Get(namespace, name string) (runtime.Object, error) {
	return d.sharedInformers.Rbac().V1().ClusterRoles().Lister().Get(name)
}

func (d *clusterrolesGetter) List(namespace string, query *query.Query) (*api.ListResult, error) {
	var roles []*rbacv1.ClusterRole
	var err error

	roles, err = d.sharedInformers.Rbac().V1().ClusterRoles().Lister().List(query.Selector())
	if err != nil {
		return nil, err
	}
	return v1alpha3.DefaultObjectList(roles, query, d.compare, d.filter), nil
}

func (d *clusterrolesGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftClusterRole, ok := left.(*rbacv1.ClusterRole)
	if !ok {
		return false
	}

	rightClusterRole, ok := right.(*rbacv1.ClusterRole)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftClusterRole.ObjectMeta, rightClusterRole.ObjectMeta, field)
}

func (d *clusterrolesGetter) filter(object runtime.Object, filter query.Filter) bool {
	role, ok := object.(*rbacv1.ClusterRole)

	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(role.ObjectMeta, filter)
}
