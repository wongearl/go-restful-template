// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned/typed/core.ai.io/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeCoreV1 struct {
	*testing.Fake
}

func (c *FakeCoreV1) Healths() v1.HealthInterface {
	return &FakeHealths{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeCoreV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}