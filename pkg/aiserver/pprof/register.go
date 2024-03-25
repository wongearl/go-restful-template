package pprof

import (
	"net/http"
	pprof "net/http/pprof"

	"github.com/emicklei/go-restful"
)

// AddToContainer adds the pprof into the conatiner
func AddToContainer(container *restful.Container) {
	ws := &restful.WebService{}
	ws.Path("/debug/pprof")
	ws.Route(ws.GET("/").To(nativeHandlerWrapper(pprof.Index)))
	ws.Route(ws.GET("/{subpath:*}").To(nativeHandlerWrapper(pprof.Index)))
	ws.Route(ws.GET("/cmdline").To(nativeHandlerWrapper(pprof.Cmdline)))
	ws.Route(ws.GET("/profile").To(nativeHandlerWrapper(pprof.Profile)))
	ws.Route(ws.GET("/symbol").To(nativeHandlerWrapper(pprof.Symbol)))
	ws.Route(ws.GET("/trace").To(nativeHandlerWrapper(pprof.Trace)))
	container.Add(ws)
}

func nativeHandlerWrapper(handler func(http.ResponseWriter, *http.Request)) restful.RouteFunction {
	return func(r1 *restful.Request, r2 *restful.Response) {
		handler(r2.ResponseWriter, r1.Request)
	}
}
