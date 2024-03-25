package loginrecord

import (
	"github.com/wongearl/go-restful-template/pkg/api"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	alinformers "github.com/wongearl/go-restful-template/pkg/client/ai/informers/externalversions"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	"k8s.io/apimachinery/pkg/runtime"
)

const recordType = "type"

type loginrecordsGetter struct {
	alInformer alinformers.SharedInformerFactory
}

func New(alinformer alinformers.SharedInformerFactory) v1alpha3.Interface {
	return &loginrecordsGetter{alInformer: alinformer}
}

func (d *loginrecordsGetter) Get(_, name string) (runtime.Object, error) {
	return d.alInformer.Iam().V1().Users().Lister().Get(name)
}

func (d *loginrecordsGetter) List(_ string, query *query.Query) (*api.ListResult, error) {
	records, err := d.alInformer.Iam().V1().LoginRecords().Lister().List(query.Selector())
	if err != nil {
		return nil, err
	}

	return v1alpha3.DefaultObjectList(records, query, d.compare, d.filter), nil
}

func (d *loginrecordsGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftUser, ok := left.(*iamv1.LoginRecord)
	if !ok {
		return false
	}

	rightUser, ok := right.(*iamv1.LoginRecord)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftUser.ObjectMeta, rightUser.ObjectMeta, field)
}

func (d *loginrecordsGetter) filter(object runtime.Object, filter query.Filter) bool {
	record, ok := object.(*iamv1.LoginRecord)

	if !ok {
		return false
	}

	switch filter.Field {
	case recordType:
		return string(record.Spec.Type) == string(filter.Value)
	default:
		return v1alpha3.DefaultObjectMetaFilter(record.ObjectMeta, filter)
	}

}
