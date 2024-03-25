package oauth

import (
	"context"
	"fmt"
	"time"

	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/oauth"
	"github.com/wongearl/go-restful-template/pkg/aiserver/config"
	"github.com/wongearl/go-restful-template/pkg/aiserver/request"
	"github.com/wongearl/go-restful-template/pkg/api"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	"github.com/wongearl/go-restful-template/pkg/constants"
	"github.com/wongearl/go-restful-template/pkg/middles/auth"
	"github.com/wongearl/go-restful-template/pkg/middles/iam/im"
	"github.com/wongearl/go-restful-template/pkg/utils/stringutils"

	restful "github.com/emicklei/go-restful"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authuser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	passwordGrantType     = "password"
	refreshTokenGrantType = "refresh_token"
)

type handler struct {
	im                    im.IdentityManagementInterface
	tokenOperator         auth.TokenManagementInterface
	passwordAuthenticator auth.PasswordAuthenticator
	loginRecorder         auth.LoginRecorder
	k8sclient             kubernetes.Interface
	option                *config.AiOptions
}

func newHandler(im im.IdentityManagementInterface,
	tokenOperator auth.TokenManagementInterface,
	passwordAuthenticator auth.PasswordAuthenticator,
	loginRecorder auth.LoginRecorder,
	option *config.AiOptions, k8sclient kubernetes.Interface) *handler {
	return &handler{im: im,
		tokenOperator:         tokenOperator,
		passwordAuthenticator: passwordAuthenticator,
		loginRecorder:         loginRecorder,
		option:                option,
		k8sclient:             k8sclient,
	}
}

func (h *handler) Token(req *restful.Request, resp *restful.Response) {

	cm, err := h.k8sclient.CoreV1().ConfigMaps(h.option.Namespace).Get(context.Background(), fmt.Sprintf("%sai-config", h.option.NamePrefix), metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	_ = stringutils.AesDecrypt(cm.Annotations[constants.RegisterAnnotationKey], constants.RegisterSecretKey)

	grantType, err := req.BodyParameter("grant_type")
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}
	switch grantType {
	case passwordGrantType:
		username, _ := req.BodyParameter("username")
		password, _ := req.BodyParameter("password")
		h.passwordGrant(username, password, req, resp)
		break
	case refreshTokenGrantType:
		h.refreshTokenGrant(req, resp)
		break
	default:
		err = apierrors.NewBadRequest(fmt.Sprintf("Grant type %s is not supported", grantType))
		api.NewEmptyResult().WithError(err).WriteTo(resp)
	}
	return
}

func (h *handler) passwordGrant(username string, password string, req *restful.Request, resp *restful.Response) {
	_, err := h.im.DescribeUser(username)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}
	authenticated, provider, err := h.passwordAuthenticator.Authenticate(username, password)
	if err != nil {
		switch err {
		case auth.IncorrectPasswordError:
			requestInfo, _ := request.RequestInfoFrom(req.Request.Context())
			if err := h.loginRecorder.RecordLogin(username, iamv1.Token, provider, requestInfo.SourceIP, requestInfo.UserAgent, err); err != nil {
				klog.Errorf("Failed to record unsuccessful login attempt for user %s, error: %v", username, err)
			}
		}
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	result, err := h.tokenOperator.IssueTo(authenticated, nil, nil)
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	requestInfo, _ := request.RequestInfoFrom(req.Request.Context())
	if err = h.loginRecorder.RecordLogin(authenticated.GetName(), iamv1.Token, provider, requestInfo.SourceIP, requestInfo.UserAgent, nil); err != nil {
		klog.Errorf("Failed to record successful login for user %s, error: %v", username, err)
	}

	api.NewResult[*oauth.Token]().WithObject(result).WithError(err).WriteTo(resp)
}

func (h *handler) refreshTokenGrant(req *restful.Request, resp *restful.Response) {
	refreshToken, err := req.BodyParameter("refresh_token")
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	authenticated, err := h.tokenOperator.Verify(refreshToken)
	if err != nil {
		err := apierrors.NewUnauthorized(fmt.Sprintf("Unauthorized: %s", err))
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}

	result, err := h.tokenOperator.IssueTo(authenticated, nil, nil)
	api.NewResult[*oauth.Token]().WithObject(result).WithError(err).WriteTo(resp)
}

func (h *handler) createToken(req *restful.Request, resp *restful.Response) {
	result, err := createToken(h.tokenOperator, req.Request.Context())
	if err != nil {
		api.NewEmptyResult().WithError(err).WriteTo(resp)
		return
	}
	api.NewResult[*oauth.Token]().WithObject(result).WithError(err).WriteTo(resp)
}

func createToken(tokenOperator auth.TokenManagementInterface, ctx context.Context) (*oauth.Token, error) {
	user := &authuser.DefaultInfo{
		Extra: map[string][]string{
			iamv1.ResourcesSingularGlobalRole: {iamv1.PlatformAdmin},
		},
	}
	if userInfo, ok := request.UserFrom(ctx); ok {
		user.Name = userInfo.GetName()
	}
	accessTokenMaxAgeSecond := time.Second * 0
	accessTokenInactivityTimeoutSecond := time.Second * 0
	return tokenOperator.IssueTo(user, &accessTokenMaxAgeSecond, &accessTokenInactivityTimeoutSecond)
}
