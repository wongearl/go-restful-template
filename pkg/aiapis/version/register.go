package version

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/runtime"
	"github.com/wongearl/go-restful-template/pkg/version"

	restful "github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func AddToContainer(container *restful.Container) error {
	ws := runtime.NewWebService(schema.GroupVersion{})

	ws.Route(ws.GET("/version").
		To(func(request *restful.Request, respone *restful.Response) {
			versionInfo := version.Get()

			respone.WriteAsJson(versionInfo)
		})).Doc("ai server version")

	container.Add(ws)
	return nil
}
