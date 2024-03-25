package globalrolebinding

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	informers "github.com/wongearl/go-restful-template/pkg/client/ai/informers/externalversions"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	"k8s.io/apimachinery/pkg/runtime"
)

type globalrolebindingsGetter struct {
	sharedInformers informers.SharedInformerFactory
}

func New(sharedInformers informers.SharedInformerFactory) v1alpha3.Interface {
	return &globalrolebindingsGetter{sharedInformers: sharedInformers}
}

func (d *globalrolebindingsGetter) Get(_, name string) (runtime.Object, error) {
	return d.sharedInformers.Iam().V1().GlobalRoleBindings().Lister().Get(name)
}

func (d *globalrolebindingsGetter) List(_ string, query *query.Query) (*api.ListResult, error) {
	globalRoleBindings, err := d.sharedInformers.Iam().V1().GlobalRoleBindings().Lister().List(query.Selector())
	if err != nil {
		return nil, err
	}

	return v1alpha3.DefaultObjectList(globalRoleBindings, query, d.compare, d.filter), nil
}

func (d *globalrolebindingsGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftRoleBinding, ok := left.(*iamv1.GlobalRoleBinding)
	if !ok {
		return false
	}

	rightRoleBinding, ok := right.(*iamv1.GlobalRoleBinding)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftRoleBinding.ObjectMeta, rightRoleBinding.ObjectMeta, field)
}

func (d *globalrolebindingsGetter) filter(object runtime.Object, filter query.Filter) bool {
	role, ok := object.(*iamv1.GlobalRoleBinding)

	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(role.ObjectMeta, filter)
}
