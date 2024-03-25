package role

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
)

type rolesGetter struct {
	sharedInformers informers.SharedInformerFactory
}

func New(sharedInformers informers.SharedInformerFactory) v1alpha3.Interface {
	return &rolesGetter{sharedInformers: sharedInformers}
}

func (d *rolesGetter) Get(namespace, name string) (runtime.Object, error) {
	return d.sharedInformers.Rbac().V1().Roles().Lister().Roles(namespace).Get(name)
}

func (d *rolesGetter) List(namespace string, query *query.Query) (*api.ListResult, error) {
	var roles []*rbacv1.Role
	var err error
	roles, err = d.sharedInformers.Rbac().V1().Roles().Lister().Roles(namespace).List(query.Selector())
	if err != nil {
		return nil, err
	}

	return v1alpha3.DefaultObjectList(roles, query, d.compare, d.filter), nil
}

func (d *rolesGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftRole, ok := left.(*rbacv1.Role)
	if !ok {
		return false
	}

	rightRole, ok := right.(*rbacv1.Role)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftRole.ObjectMeta, rightRole.ObjectMeta, field)
}

func (d *rolesGetter) filter(object runtime.Object, filter query.Filter) bool {
	role, ok := object.(*rbacv1.Role)

	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(role.ObjectMeta, filter)
}
