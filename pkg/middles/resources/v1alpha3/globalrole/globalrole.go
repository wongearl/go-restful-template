package globalrole

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	informers "github.com/wongearl/go-restful-template/pkg/client/ai/informers/externalversions"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	"k8s.io/apimachinery/pkg/runtime"
)

type globalrolesGetter struct {
	sharedInformers informers.SharedInformerFactory
}

func New(sharedInformers informers.SharedInformerFactory) v1alpha3.Interface {
	return &globalrolesGetter{sharedInformers: sharedInformers}
}

func (d *globalrolesGetter) Get(_, name string) (runtime.Object, error) {
	return d.sharedInformers.Iam().V1().GlobalRoles().Lister().Get(name)
}

func (d *globalrolesGetter) List(_ string, query *query.Query) (*api.ListResult, error) {
	var roles []*iamv1.GlobalRole
	var err error

	roles, err = d.sharedInformers.Iam().V1().GlobalRoles().Lister().List(query.Selector())
	if err != nil {
		return nil, err
	}

	return v1alpha3.DefaultObjectList(roles, query, d.compare, d.filter), nil
}

func (d *globalrolesGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftRole, ok := left.(*iamv1.GlobalRole)
	if !ok {
		return false
	}

	rightRole, ok := right.(*iamv1.GlobalRole)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftRole.ObjectMeta, rightRole.ObjectMeta, field)
}

func (d *globalrolesGetter) filter(object runtime.Object, filter query.Filter) bool {
	role, ok := object.(*iamv1.GlobalRole)

	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(role.ObjectMeta, filter)
}
