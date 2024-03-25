package k8s

import (
	aiclient "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned"
	apiExtensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client interface {
	Kubernetes() kubernetes.Interface
	ApiExtensions() apiExtensionsclient.Interface
	Ai() aiclient.Interface
	Discovery() discovery.DiscoveryInterface
	Master() string
	Config() *rest.Config
}

type kubernetesClient struct {
	k8s kubernetes.Interface

	discoveryClient *discovery.DiscoveryClient

	apiExtensions apiExtensionsclient.Interface

	ai aiclient.Interface

	master string

	config *rest.Config
}

func NewKubernetesClient(options *KubernetesOptions) (Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", options.KubeConfig)
	if err != nil {
		return nil, err
	}

	config.QPS = options.QPS
	config.Burst = options.Burst
	if options.Token != "" {
		// override the token
		config = &rest.Config{
			Host: config.Host,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
			BearerToken: options.Token,
		}
	}

	var k kubernetesClient
	k.k8s, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	k.discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	k.apiExtensions, err = apiExtensionsclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	k.ai, err = aiclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	k.master = options.Master
	k.config = config

	return &k, nil

}

func (k *kubernetesClient) Kubernetes() kubernetes.Interface {
	return k.k8s
}

func (k *kubernetesClient) Discovery() discovery.DiscoveryInterface {
	return k.discoveryClient
}

func (k *kubernetesClient) ApiExtensions() apiExtensionsclient.Interface {
	return k.apiExtensions
}

func (k *kubernetesClient) Ai() aiclient.Interface {
	return k.ai
}

func (k *kubernetesClient) Master() string {
	return k.master
}

func (k *kubernetesClient) Config() *rest.Config {
	return k.config
}
