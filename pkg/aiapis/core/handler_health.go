package core

import (
	"github.com/wongearl/go-restful-template/pkg/api"
	corev1 "github.com/wongearl/go-restful-template/pkg/api/core.ai.io/v1"
	coretypedv1 "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned/typed/core.ai.io/v1"

	restful "github.com/emicklei/go-restful"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type healthHandler struct {
	cacheClient coretypedv1.CoreV1Interface
}

func (h *healthHandler) list(req *restful.Request, resp *restful.Response) {
	healths, err := h.cacheClient.Healths().List(req.Request.Context(), metav1.ListOptions{})
	api.NewResult[corev1.Health]().WithList(healths.Items).WithError(err).WriteTo(resp)
}

func (h *healthHandler) getHealth(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter(healthParamKey.Data().Name)
	health, err := h.cacheClient.Healths().Get(req.Request.Context(), name, metav1.GetOptions{})
	api.NewResult[*corev1.Health]().WithObject(health).WithError(err).WriteTo(resp)
}
