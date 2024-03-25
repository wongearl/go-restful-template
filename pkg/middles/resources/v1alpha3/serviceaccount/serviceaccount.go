package serviceaccount

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"

	"github.com/wongearl/go-restful-template/pkg/api"
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"
)

type serviceaccountsGetter struct {
	informer informers.SharedInformerFactory
}

func New(sharedInformers informers.SharedInformerFactory) v1alpha3.Interface {
	return &serviceaccountsGetter{informer: sharedInformers}
}

func (d *serviceaccountsGetter) Get(namespace, name string) (runtime.Object, error) {
	return d.informer.Core().V1().ServiceAccounts().Lister().ServiceAccounts(namespace).Get(name)
}

func (d *serviceaccountsGetter) List(namespace string, query *query.Query) (*api.ListResult, error) {
	serviceaccounts, err := d.informer.Core().V1().ServiceAccounts().Lister().ServiceAccounts(namespace).List(query.Selector())
	if err != nil {
		return nil, err
	}

	return v1alpha3.DefaultObjectList(serviceaccounts, query, d.compare, d.filter), nil
}

func (d *serviceaccountsGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftCM, ok := left.(*corev1.ServiceAccount)
	if !ok {
		return false
	}

	rightCM, ok := right.(*corev1.ServiceAccount)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftCM.ObjectMeta, rightCM.ObjectMeta, field)
}

func (d *serviceaccountsGetter) filter(object runtime.Object, filter query.Filter) bool {
	serviceAccount, ok := object.(*corev1.ServiceAccount)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(serviceAccount.ObjectMeta, filter)
}
