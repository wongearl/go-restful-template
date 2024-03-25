package v1

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/authorization/authorizer"
	"github.com/wongearl/go-restful-template/pkg/aiserver/config"
	"github.com/wongearl/go-restful-template/pkg/aiserver/runtime"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	"github.com/wongearl/go-restful-template/pkg/middles/iam/am"
	"github.com/wongearl/go-restful-template/pkg/middles/iam/im"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GroupName = "iam.ai.io"
)

var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1"}

func AddToContainer(container *restful.Container, im im.IdentityManagementInterface, am am.AccessManagementInterface, option *config.AiOptions, authorizer authorizer.Authorizer) error {
	ws := runtime.NewWebService(GroupVersion)
	handler := newIAMHandler(im, am, option, authorizer)

	// users
	ws.Route(ws.POST("/users").
		To(handler.CreateUser).
		Reads(iamv1.User{}).
		Doc("Create a global user account."))
	ws.Route(ws.DELETE("/users/{user}").
		To(handler.DeleteUser).
		Param(ws.PathParameter("user", "username")).
		Doc("Delete the specified user."))
	ws.Route(ws.PUT("/users/{user}").
		To(handler.UpdateUser).
		Reads(iamv1.User{}).
		Param(ws.PathParameter("user", "username")).
		Doc("Update user profile."))
	ws.Route(ws.PUT("/users/{user}/password").
		To(handler.ModifyPassword).
		Reads(PasswordReset{}).
		Param(ws.PathParameter("user", "username")).
		Doc("Reset password of the specified user."))
	ws.Route(ws.GET("/users/{user}").
		To(handler.DescribeUser).
		Param(ws.PathParameter("user", "username")).
		Doc("Retrieve user details."))
	ws.Route(ws.GET("/users/{user}/details").
		To(handler.GetUserDetail).
		Doc("Get a user detail.").
		Param(ws.PathParameter("user", "username")))
	ws.Route(ws.GET("/users").
		To(handler.ListUsers).
		Doc("List all users."))

	// loginrecords
	ws.Route(ws.GET("/users/{user}/loginrecords").
		To(handler.ListUserLoginRecords).
		Param(ws.PathParameter("user", "username of the user")).
		Doc("List login records of the specified user."))

	// namespacemembers
	ws.Route(ws.GET("/namespaces/{namespace}/members").
		To(handler.ListNamespaceMembers).
		Param(ws.PathParameter("namespace", "namespace")).
		Doc("List all members in the specified namespace."))
	ws.Route(ws.GET("/namespaces/{namespace}/members/{member}").
		To(handler.DescribeNamespaceMember).
		Param(ws.PathParameter("namespace", "namespace")).
		Param(ws.PathParameter("member", "namespace member's username")).
		Doc("Retrieve the role of the specified member."))
	ws.Route(ws.POST("/namespaces/{namespace}/members").
		To(handler.CreateNamespaceMembers).
		Reads([]Member{}).
		Param(ws.PathParameter("namespace", "namespace")).
		Doc("Add members to the namespace in bulk."))
	ws.Route(ws.PUT("/namespaces/{namespace}/members/{member}").
		To(handler.UpdateNamespaceMember).
		Reads(Member{}).
		Param(ws.PathParameter("namespace", "namespace")).
		Param(ws.PathParameter("member", "namespace member's username")).
		Doc("Update the role bind of the member."))
	ws.Route(ws.DELETE("/namespaces/{namespace}/members/{member}").
		To(handler.RemoveNamespaceMember).
		Param(ws.PathParameter("namespace", "namespace")).
		Param(ws.PathParameter("member", "namespace member's username")).
		Doc("Delete a member from the namespace."))

	// globalroles
	ws.Route(ws.GET("/globalroles").
		To(handler.ListGlobalRoles).
		Doc("List all global roles."))

	// roles
	ws.Route(ws.GET("/namespaces/{namespace}/roles").
		To(handler.ListRoles).
		Param(ws.PathParameter("namespace", "namespace")).
		Doc("List all roles in the specified namespace."))

	container.Add(ws)
	return nil
}
