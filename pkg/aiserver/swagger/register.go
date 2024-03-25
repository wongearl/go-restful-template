package swagger

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"

	restful "github.com/emicklei/go-restful"
	"k8s.io/klog"
)

const rootPath = "/apidocs"

// AddToContainer adds swagger-ui
func AddToContainer(rootDir string, container *restful.Container) {
	ws := &restful.WebService{}
	ws.Path(rootPath)
	ws.Route(ws.GET("/{subpath:*}").To(func(req *restful.Request, resp *restful.Response) {
		actual := path.Join(rootDir, req.PathParameter("subpath"))
		http.ServeFile(
			resp.ResponseWriter,
			req.Request,
			actual)
	}))
	ws.Route(ws.GET("/").To(func(req *restful.Request, resp *restful.Response) {
		scheme := req.Request.URL.Scheme
		var host string

		if val := req.HeaderParameter("X-Real-IP"); val != "" {
			host = val
			klog.V(7).Infof("get host '%s' from header 'X-Real-IP'", val)
		} else {
			host = req.Request.Host
		}

		proto := req.HeaderParameter("X-Forwarded-Proto")
		if scheme == "" && proto != "" {
			scheme = proto
		} else {
			scheme = "http"
		}

		url := req.QueryParameter("url")
		if url == "" {
			http.Redirect(resp.ResponseWriter, req.Request, fmt.Sprintf("%s/?url=%s://%s/apidocs.json", rootPath, scheme, host), http.StatusPermanentRedirect)
		} else {
			absDir, err := filepath.Abs(rootDir)
			if err != nil {
				resp.WriteError(http.StatusBadGateway, err)
				return
			}

			actual := path.Join(absDir, "index.html")
			http.ServeFile(
				resp.ResponseWriter,
				req.Request,
				actual)
		}
	}))
	container.Add(ws)
}
