package clusterclient

import (
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type clusterClients struct {
	sync.RWMutex
}

type ClusterClients interface {
	GetClusterKubeconfig(string) (string, error)
	GetKubebernetsClientSet(string) (*kubernetes.Clientset, error)
}

func (c *clusterClients) GetKubebernetsClientSet(kubeconfig string) (*kubernetes.Clientset, error) {

	restConfig, err := newRestConfigFromString(kubeconfig)
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return clientSet, nil
}

func newRestConfigFromString(kubeconfig string) (*rest.Config, error) {
	bytes, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}
	return bytes.ClientConfig()
}
