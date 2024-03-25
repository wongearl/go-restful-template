package runtime

import (
	restful "github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	ApiRootPath = "/ai-apis"
)

const (
	MimeMergePatchJson = "application/merge-patch+json"
	MimeJsonPatchJson  = "application/json-patch+json"
)

func init() {
	restful.RegisterEntityAccessor(MimeMergePatchJson, restful.NewEntityAccessorJSON(restful.MIME_JSON))
	restful.RegisterEntityAccessor(MimeJsonPatchJson, restful.NewEntityAccessorJSON(restful.MIME_JSON))
}

func NewWebService(gv schema.GroupVersion) *restful.WebService {
	webservice := restful.WebService{}
	webservice.Path(ApiRootPath + "/" + gv.String()).Produces(restful.MIME_JSON)

	return &webservice
}
