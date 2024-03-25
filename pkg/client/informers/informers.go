package informers

import (
	"reflect"
	"time"

	aiclient "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned"
	aiinformers "github.com/wongearl/go-restful-template/pkg/client/ai/informers/externalversions"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	apiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	"k8s.io/client-go/informers"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

const defaultResync = 600 * time.Second

type InformerFactory interface {
	KubernetesSharedInformerFactory() informers.SharedInformerFactory
	ApiExtensionSharedInformerFactory() externalversions.SharedInformerFactory
	AiSharedInformerFactory() aiinformers.SharedInformerFactory

	Start(stopCh <-chan struct{})
}

type GenericInformerFactory interface {
	Start(stopCh <-chan struct{})
	WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
}

type informerFactories struct {
	informerFactory              informers.SharedInformerFactory
	apiextensionsInformerFactory externalversions.SharedInformerFactory
	alSharedInformerFactory      aiinformers.SharedInformerFactory
}

func NewInformerFactories(client kubernetes.Interface, apiextensionsClient apiextensionsclient.Interface, aiClient aiclient.Interface) InformerFactory {
	factory := &informerFactories{}

	if client != nil {
		factory.informerFactory = k8sinformers.NewSharedInformerFactory(client, defaultResync)
	}

	if apiextensionsClient != nil {
		factory.apiextensionsInformerFactory = apiextensionsinformers.NewSharedInformerFactory(apiextensionsClient, defaultResync)
	}

	if aiClient != nil {
		factory.alSharedInformerFactory = aiinformers.NewSharedInformerFactory(aiClient, defaultResync)
	}

	return factory
}

func (f *informerFactories) KubernetesSharedInformerFactory() informers.SharedInformerFactory {
	return f.informerFactory
}

func (f *informerFactories) ApiExtensionSharedInformerFactory() externalversions.SharedInformerFactory {
	return f.apiextensionsInformerFactory
}

func (f *informerFactories) AiSharedInformerFactory() aiinformers.SharedInformerFactory {
	return f.alSharedInformerFactory
}

func (f *informerFactories) Start(stopCh <-chan struct{}) {
	if f.informerFactory != nil {
		f.informerFactory.Start(stopCh)
	}

	if f.apiextensionsInformerFactory != nil {
		f.apiextensionsInformerFactory.Start(stopCh)
	}

	if f.alSharedInformerFactory != nil {
		f.alSharedInformerFactory.Start(stopCh)
	}
}
