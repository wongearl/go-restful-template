package auth

import (
	"context"
	"fmt"
	"net/mail"

	authoptions "github.com/wongearl/go-restful-template/pkg/aiserver/authentication/options"
	"github.com/wongearl/go-restful-template/pkg/aiserver/config"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	ai "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned"
	iamv1listers "github.com/wongearl/go-restful-template/pkg/client/ai/listers/iam.ai.io/v1"

	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	authuser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog"
)

var (
	RateLimitExceededError  = fmt.Errorf("auth rate limit exceeded")
	IncorrectPasswordError  = fmt.Errorf("incorrect password")
	AccountIsNotActiveError = fmt.Errorf("account is not active")
)

type PasswordAuthenticator interface {
	Authenticate(username, password string) (authuser.Info, string, error)
}

type passwordAuthenticator struct {
	aiClient    ai.Interface
	userGetter  *userGetter
	authOptions *authoptions.AuthenticationOptions
	alOptions   *config.AiOptions
}

type userGetter struct {
	userLister iamv1listers.UserLister
}

func NewPasswordAuthenticator(aiClient ai.Interface,
	userLister iamv1listers.UserLister,
	authOptions *authoptions.AuthenticationOptions,
	alOptions *config.AiOptions) PasswordAuthenticator {
	passwordAuthenticator := &passwordAuthenticator{
		aiClient:    aiClient,
		userGetter:  &userGetter{userLister: userLister},
		authOptions: authOptions,
		alOptions:   alOptions,
	}
	return passwordAuthenticator
}

func (p *passwordAuthenticator) Authenticate(username, password string) (authuser.Info, string, error) {
	// empty username or password are not allowed
	if username == "" || password == "" {
		return nil, "", IncorrectPasswordError
	}

	// account
	user, err := p.userGetter.findUser(username)
	if err != nil {
		// ignore not found error
		if !errors.IsNotFound(err) {
			klog.Error(err)
			return nil, "", err
		}
	}

	// check user status
	if user != nil && (user.Status.State == nil || *user.Status.State != iamv1.UserActive) {
		if user.Status.State != nil && *user.Status.State == iamv1.UserAuthLimitExceeded {
			klog.Errorf("%s, username: %s", RateLimitExceededError, username)
			return nil, "", RateLimitExceededError
		} else {
			// state not active
			klog.Errorf("%s, username: %s", AccountIsNotActiveError, username)
			return nil, "", AccountIsNotActiveError
		}
	}

	// if the password is not empty, means that the password has been reset, even if the user was mapping from IDP
	if user != nil && user.Spec.EncryptedPassword != "" {
		if err = PasswordVerify(user.Spec.EncryptedPassword, password); err != nil {
			klog.Error(err)
			return nil, "", err
		}
		globalrole := p.findGlobalRole(username)
		u := &authuser.DefaultInfo{
			Name: user.Name,
			Extra: map[string][]string{
				iamv1.ResourcesSingularGlobalRole: {globalrole},
			},
		}
		return u, "", nil
	}

	return nil, "", IncorrectPasswordError
}

func PasswordVerify(encryptedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(encryptedPassword), []byte(password)); err != nil {
		return IncorrectPasswordError
	}
	return nil
}

// findUser
func (u *userGetter) findUser(username string) (*iamv1.User, error) {
	if _, err := mail.ParseAddress(username); err != nil {
		return u.userLister.Get(username)
	} else {
		users, err := u.userLister.List(labels.Everything())
		if err != nil {
			klog.Error(err)
			return nil, err
		}
		for _, find := range users {
			if find.Spec.Email == username {
				return find, nil
			}
		}
	}

	return nil, errors.NewNotFound(iamv1.Resource("user"), username)
}

func (p *passwordAuthenticator) findGlobalRole(name string) string {
	if name == iamv1.NamespaceAdmin {
		return iamv1.PlatformAdmin
	}
	var err error
	_, err = p.aiClient.IamV1().GlobalRoleBindings().Get(context.Background(), name+"-"+p.alOptions.NamePrefix+iamv1.PlatformAdmin, metav1.GetOptions{})
	if err == nil {
		return iamv1.PlatformAdmin
	}
	_, err = p.aiClient.IamV1().GlobalRoleBindings().Get(context.Background(), name+"-"+p.alOptions.NamePrefix+iamv1.PlatformSelfProvisioner, metav1.GetOptions{})
	if err == nil {
		return iamv1.PlatformSelfProvisioner
	}
	_, err = p.aiClient.IamV1().GlobalRoleBindings().Get(context.Background(), name+"-"+p.alOptions.NamePrefix+iamv1.PlatformRegular, metav1.GetOptions{})
	if err == nil {
		return iamv1.PlatformRegular
	}
	return ""
}
