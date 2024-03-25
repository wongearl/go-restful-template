package rolebinding

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
)

type rolebindingsGetter struct {
	sharedInformers informers.SharedInformerFactory
}

func New(sharedInformers informers.SharedInformerFactory) v1alpha3.Interface {
	return &rolebindingsGetter{sharedInformers: sharedInformers}
}

func (d *rolebindingsGetter) Get(namespace, name string) (runtime.Object, error) {
	return d.sharedInformers.Rbac().V1().RoleBindings().Lister().RoleBindings(namespace).Get(name)
}

func (d *rolebindingsGetter) List(namespace string, query *query.Query) (*api.ListResult, error) {
	roleBindings, err := d.sharedInformers.Rbac().V1().RoleBindings().Lister().RoleBindings(namespace).List(query.Selector())
	if err != nil {
		return nil, err
	}

	return v1alpha3.DefaultObjectList(roleBindings, query, d.compare, d.filter), nil
}

func (d *rolebindingsGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftRoleBinding, ok := left.(*rbacv1.RoleBinding)
	if !ok {
		return false
	}

	rightRoleBinding, ok := right.(*rbacv1.RoleBinding)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftRoleBinding.ObjectMeta, rightRoleBinding.ObjectMeta, field)
}

func (d *rolebindingsGetter) filter(object runtime.Object, filter query.Filter) bool {
	role, ok := object.(*rbacv1.RoleBinding)

	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(role.ObjectMeta, filter)
}
