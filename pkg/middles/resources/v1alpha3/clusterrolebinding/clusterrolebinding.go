package clusterrolebinding

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
)

type clusterrolebindingsGetter struct {
	sharedInformers informers.SharedInformerFactory
}

func New(sharedInformers informers.SharedInformerFactory) v1alpha3.Interface {
	return &clusterrolebindingsGetter{sharedInformers: sharedInformers}
}

func (d *clusterrolebindingsGetter) Get(_, name string) (runtime.Object, error) {
	return d.sharedInformers.Rbac().V1().ClusterRoleBindings().Lister().Get(name)
}

func (d *clusterrolebindingsGetter) List(_ string, query *query.Query) (*api.ListResult, error) {
	roleBindings, err := d.sharedInformers.Rbac().V1().ClusterRoleBindings().Lister().List(query.Selector())
	if err != nil {
		return nil, err
	}

	return v1alpha3.DefaultObjectList(roleBindings, query, d.compare, d.filter), nil
}

func (d *clusterrolebindingsGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftRoleBinding, ok := left.(*rbacv1.ClusterRoleBinding)
	if !ok {
		return false
	}

	rightRoleBinding, ok := right.(*rbacv1.ClusterRoleBinding)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftRoleBinding.ObjectMeta, rightRoleBinding.ObjectMeta, field)
}

func (d *clusterrolebindingsGetter) filter(object runtime.Object, filter query.Filter) bool {
	role, ok := object.(*rbacv1.ClusterRoleBinding)

	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(role.ObjectMeta, filter)
}
