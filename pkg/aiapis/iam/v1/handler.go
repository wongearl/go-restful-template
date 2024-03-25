package v1

import (
	"fmt"
	"strings"

	"github.com/wongearl/go-restful-template/pkg/aiserver/authorization/authorizer"
	"github.com/wongearl/go-restful-template/pkg/aiserver/config"
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	apirequest "github.com/wongearl/go-restful-template/pkg/aiserver/request"
	"github.com/wongearl/go-restful-template/pkg/api"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	"github.com/wongearl/go-restful-template/pkg/middles/auth"
	"github.com/wongearl/go-restful-template/pkg/middles/iam/am"
	"github.com/wongearl/go-restful-template/pkg/middles/iam/im"
	"github.com/wongearl/go-restful-template/pkg/utils/stringutils"

	restful "github.com/emicklei/go-restful"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	authuser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog"
)

type Member struct {
	Username string `json:"username"`
	RoleRef  string `json:"roleRef"`
}

type PasswordReset struct {
	CurrentPassword string `json:"currentPassword"`
	Password        string `json:"password"`
}

type UserDetails struct {
	Username       string            `json:"username"`
	Email          string            `json:"email"`
	GlobalRole     string            `json:"globalrole"`
	WorkspaceRoles map[string]string `json:"workspaceroles"`
	NamespaceRoles map[string]string `json:"namespaceroles"`
}

type iamHandler struct {
	am         am.AccessManagementInterface
	im         im.IdentityManagementInterface
	option     *config.AiOptions
	authorizer authorizer.Authorizer
}

func newIAMHandler(im im.IdentityManagementInterface, am am.AccessManagementInterface, option *config.AiOptions, authorizer authorizer.Authorizer) *iamHandler {
	return &iamHandler{
		am:         am,
		im:         im,
		option:     option,
		authorizer: authorizer,
	}
}

func (h *iamHandler) DescribeUser(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("user")
	user, err := h.im.DescribeUser(username)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	globalRole, err := h.am.GetGlobalRoleOfUser(username)
	if err != nil && !errors.IsNotFound(err) {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}
	if globalRole != nil {
		user = appendGlobalRoleAnnotation(user, stringutils.ReplaceStringOnce(globalRole.Name, h.option.NamePrefix))
	}
	api.NewResult[*iamv1.User]().WithObject(user).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) GetUserDetail(req *restful.Request, resp *restful.Response) {
	var err error
	userDetail := &UserDetails{
		WorkspaceRoles: make(map[string]string, 0),
		NamespaceRoles: make(map[string]string, 0),
	}

	username := req.PathParameter("user")
	userCrd, err := h.im.DescribeUser(username)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}
	userDetail.Username = userCrd.Name
	userDetail.Email = userCrd.Spec.Email

	globalRole, err := h.am.GetGlobalRoleOfUser(username)
	// ignore not found error
	if err != nil && !errors.IsNotFound(err) {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}
	userDetail.GlobalRole = stringutils.ReplaceStringOnce(globalRole.Name, h.option.NamePrefix)
	api.NewResult[UserDetails]().WithObject(*userDetail).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) ListUsers(req *restful.Request, resp *restful.Response) {
	queryParam := query.ParseQueryParameter(req)
	result, err := h.im.ListUsers(queryParam)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}
	for i, item := range result.Items {
		user := &item
		user = user.DeepCopy()
		globalRole, err := h.am.GetGlobalRoleOfUser(user.Name)
		if err != nil && !errors.IsNotFound(err) {
			api.NewEmptyResult().WithError(err).WriteTo(resp)
			return
		}
		if globalRole != nil {
			user = appendGlobalRoleAnnotation(user, stringutils.ReplaceStringOnce(globalRole.Name, h.option.NamePrefix))
		}

		result.Items[i] = *user
	}
	api.NewResult[iamv1.User]().WithListAndFilter(result.Items, req).WithError(err).WriteTo(resp)
	return
}

func appendGlobalRoleAnnotation(user *iamv1.User, globalRole string) *iamv1.User {
	if user.Annotations == nil {
		user.Annotations = make(map[string]string, 0)
	}
	user.Annotations[iamv1.GlobalRoleAnnotation] = globalRole
	return user
}

func (h *iamHandler) ListRoles(req *restful.Request, resp *restful.Response) {
	namespace := req.PathParameter("namespace")

	queryParam := query.ParseQueryParameter(req)
	result, err := h.am.ListRoles(namespace, queryParam)
	api.NewResult[rbacv1.Role]().WithListAndFilter(result.Items, req).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) ListGlobalRoles(req *restful.Request, resp *restful.Response) {
	queryParam := query.ParseQueryParameter(req)
	result, err := h.am.ListGlobalRoles(queryParam)
	api.NewResult[iamv1.GlobalRole]().WithListAndFilter(result.Items, req).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) ListNamespaceMembers(req *restful.Request, resp *restful.Response) {
	queryParam := query.ParseQueryParameter(req)
	namespace := req.PathParameter("namespace")

	queryParam.Filters[iamv1.ScopeNamespace] = query.Value(namespace)
	result, err := h.im.ListUsers(queryParam)
	api.NewResult[iamv1.User]().WithListAndFilter(result.Items, req).WithError(err).WriteTo(resp)
	return

}

func (h *iamHandler) DescribeNamespaceMember(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("member")
	namespace := req.PathParameter("namespace")

	queryParam := query.New()
	queryParam.Filters[query.FieldNames] = query.Value(username)
	queryParam.Filters[iamv1.ScopeNamespace] = query.Value(namespace)

	result, err := h.im.ListUsers(queryParam)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	if len(result.Items) == 0 {
		err := errors.NewNotFound(iamv1.Resource(iamv1.ResourcesSingularUser), username)
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	api.NewResult[iamv1.User]().WithObject(result.Items[0]).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) CreateUser(req *restful.Request, resp *restful.Response) {
	var user iamv1.User
	err := req.ReadEntity(&user)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	globalRole := user.Annotations[iamv1.GlobalRoleAnnotation]
	delete(user.Annotations, iamv1.GlobalRoleAnnotation)

	created, err := h.im.CreateUser(&user)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	if err = h.am.CreateGlobalRoleBinding(user.Name, h.option.NamePrefix+globalRole); err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	// ensure encrypted password will not be output
	created.Spec.EncryptedPassword = ""
	api.NewResult[*iamv1.User]().WithObject(created).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) UpdateUser(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("user")

	var user iamv1.User
	err := req.ReadEntity(&user)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	if username != user.Name {
		err := fmt.Errorf("the name of the object (%s) does not match the name on the URL (%s)", user.Name, username)
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	globalRole := user.Annotations[iamv1.GlobalRoleAnnotation]
	delete(user.Annotations, iamv1.GlobalRoleAnnotation)

	updated, err := h.im.UpdateUser(&user)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	operator, ok := apirequest.UserFrom(req.Request.Context())
	if globalRole != "" && ok {
		err = h.updateGlobalRoleBinding(operator, updated, h.option.NamePrefix+globalRole)
		if err != nil {
			api.NewEmptyResult().WithError(err).WriteTo(resp)
			return
		}
		updated = appendGlobalRoleAnnotation(updated, globalRole)
	}

	api.NewResult[*iamv1.User]().WithObject(updated).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) ModifyPassword(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("user")
	var passwordReset PasswordReset
	err := req.ReadEntity(&passwordReset)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	operator, ok := apirequest.UserFrom(req.Request.Context())
	if !ok {
		err = errors.NewInternalError(fmt.Errorf("cannot obtain user info"))
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	userManagement := authorizer.AttributesRecord{
		Resource:        "users/password",
		Verb:            "update",
		ResourceScope:   apirequest.GlobalScope,
		ResourceRequest: true,
		User:            operator,
	}

	decision, _, err := h.authorizer.Authorize(userManagement)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	if decision != authorizer.DecisionAllow || passwordReset.CurrentPassword != "" {
		if err = h.im.PasswordVerify(username, passwordReset.CurrentPassword); err != nil {
			if err == auth.IncorrectPasswordError {
				err = errors.NewBadRequest("incorrect old password")
			}
			api.NewEmptyResult().WithError(err).WriteTo(resp)
			return
		}
	}

	err = h.im.ModifyPassword(username, passwordReset.Password)
	api.NewEmptyResult().WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) DeleteUser(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("user")

	operator, ok := apirequest.UserFrom(req.Request.Context())
	if !ok {
		err := errors.NewInternalError(fmt.Errorf("cannot obtain user info"))
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}
	if username == operator.GetName() {
		api.NewEmptyResult().WithError(errors.NewInternalError(fmt.Errorf("用户：" + username + "无法删除，需要其他管理员删除该用户！"))).WriteTo(resp)
		return
	}

	nsRoleBindings, err := h.am.GetRoleBindingOfUser(username)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	if len(nsRoleBindings) > 0 {
		if strings.Contains(nsRoleBindings[0].Name, "-admin") {
			api.NewEmptyResult().WithError(errors.NewInternalError(fmt.Errorf("用户：" + username + "是项目:" + nsRoleBindings[0].Namespace + "的管理员，需解除项目才能删除用户！"))).WriteTo(resp)
			return
		}
	}

	err = h.im.DeleteUser(username)
	api.NewEmptyResult().WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) CreateNamespaceMembers(req *restful.Request, resp *restful.Response) {

	namespace := req.PathParameter("namespace")

	var members []Member
	err := req.ReadEntity(&members)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	for _, member := range members {
		err := h.am.CreateNamespaceRoleBinding(member.Username, namespace, member.RoleRef)
		if err != nil {
			api.NewEmptyResult().WithError(err).WriteTo(resp)
			return
		}
	}

	api.NewResult[Member]().WithListAndFilter(members, req).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) UpdateNamespaceMember(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("member")
	namespace := req.PathParameter("namespace")

	var member Member
	err := req.ReadEntity(&member)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	if username != member.Username {
		err := fmt.Errorf("the name of the object (%s) does not match the name on the URL (%s)", member.Username, username)
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	err = h.am.CreateNamespaceRoleBinding(member.Username, namespace, member.RoleRef)
	api.NewResult[Member]().WithObject(member).WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) RemoveNamespaceMember(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("member")
	namespace := req.PathParameter("namespace")

	err := h.am.RemoveUserFromNamespace(username, namespace)
	api.NewEmptyResult().WithError(err).WriteTo(resp)
	return
}

func (h *iamHandler) updateGlobalRoleBinding(operator authuser.Info, user *iamv1.User, globalRole string) error {

	oldGlobalRole, err := h.am.GetGlobalRoleOfUser(user.Name)
	if err != nil && !errors.IsNotFound(err) {
		klog.Error(err)
		return err
	}

	if oldGlobalRole != nil && oldGlobalRole.Name == globalRole {
		return nil
	}

	userManagement := authorizer.AttributesRecord{
		Resource:        iamv1.ResourcesPluralUser,
		Verb:            "update",
		ResourceScope:   apirequest.GlobalScope,
		ResourceRequest: true,
		User:            operator,
	}
	decision, _, err := h.authorizer.Authorize(userManagement)
	if err != nil {
		klog.Error(err)
		return err
	}
	if decision != authorizer.DecisionAllow {
		err = errors.NewForbidden(iamv1.Resource(iamv1.ResourcesSingularUser),
			user.Name, fmt.Errorf("update global role binding is not allowed"))
		klog.Warning(err)
		return err
	}
	if err := h.am.CreateGlobalRoleBinding(user.Name, globalRole); err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

func (h *iamHandler) ListUserLoginRecords(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("user")
	queryParam := query.ParseQueryParameter(req)
	result, err := h.im.ListLoginRecords(username, queryParam)
	api.NewResult[iamv1.LoginRecord]().WithListAndFilter(result.Items, req).WithError(err).WriteTo(resp)
	return
}
