package oauth

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/config"
	"github.com/wongearl/go-restful-template/pkg/middles/auth"
	"github.com/wongearl/go-restful-template/pkg/middles/iam/im"
	"k8s.io/client-go/kubernetes"

	"github.com/emicklei/go-restful"
)

func AddToContainer(c *restful.Container, im im.IdentityManagementInterface, option *config.AiOptions, k8sclient kubernetes.Interface,
	tokenOperator auth.TokenManagementInterface,
	passwordAuthenticator auth.PasswordAuthenticator,
	loginRecorder auth.LoginRecorder) error {

	ws := &restful.WebService{}
	ws.Path("/oauth").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	handler := newHandler(im, tokenOperator, passwordAuthenticator, loginRecorder, option, k8sclient)
	ws.Route(ws.POST("/token").
		To(handler.Token).
		Consumes("application/x-www-form-urlencoded").
		Doc("The resource owner password credentials grant type is suitable in\n" +
			"cases where the resource owner has a trust relationship with the\n" +
			"client, such as the device operating system or a highly privileged application.").
		Param(ws.FormParameter("grant_type", "Value MUST be set to \"password\".").Required(true)).
		Param(ws.FormParameter("username", "The resource owner username.").Required(true)).
		Param(ws.FormParameter("password", "The resource owner password.").Required(true)).
		Doc("The resource owner password credentials grant type is suitable in\n" +
			"cases where the resource owner has a trust relationship with the\n" +
			"client, such as the device operating system or a highly privileged application."))

	ws.Route(ws.POST("/statictoken").
		To(handler.createToken).
		Consumes("application/x-www-form-urlencoded").
		Doc("Get access token"))

	c.Add(ws)

	return nil
}
