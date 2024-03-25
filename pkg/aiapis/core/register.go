package core

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/runtime"
	corev1 "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned/typed/core.ai.io/v1"

	restful "github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// AddToContainer add APIs to the parent container
func AddToContainer(container *restful.Container, cacheClient corev1.CoreV1Interface) {
	groupVersion := schema.GroupVersion{Group: "core.ai.io", Version: "v1"}
	ws := runtime.NewWebService(groupVersion)

	healthHandlerInstance := &healthHandler{
		cacheClient: cacheClient,
	}

	ws.Route(ws.GET("/healths").
		To(healthHandlerInstance.list).
		Doc("list all the healths"))
	ws.Route(ws.GET("/healths/{health}").
		Param(healthParamKey).
		To(healthHandlerInstance.getHealth).
		Doc("Get a singal health"))
	container.Add(ws)
}

var (
	healthParamKey = restful.PathParameter("health", "health name")
)
