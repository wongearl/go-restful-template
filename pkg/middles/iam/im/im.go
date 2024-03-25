package im

import (
	"context"
	"fmt"

	authoptions "github.com/wongearl/go-restful-template/pkg/aiserver/authentication/options"
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	ai "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned"
	"github.com/wongearl/go-restful-template/pkg/middles/auth"
	resources "github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type IdentityManagementInterface interface {
	CreateUser(user *iamv1.User) (*iamv1.User, error)
	ListUsers(query *query.Query) (*iamv1.UserList, error)
	DeleteUser(username string) error
	UpdateUser(user *iamv1.User) (*iamv1.User, error)
	DescribeUser(username string) (*iamv1.User, error)
	ModifyPassword(username string, password string) error
	ListLoginRecords(username string, query *query.Query) (*iamv1.LoginRecordList, error)
	PasswordVerify(username string, password string) error
}

func NewOperator(aiClient ai.Interface, userGetter resources.Interface, loginRecordGetter resources.Interface, options *authoptions.AuthenticationOptions) IdentityManagementInterface {
	im := &imOperator{
		aiClient:          aiClient,
		userGetter:        userGetter,
		loginRecordGetter: loginRecordGetter,
		options:           options,
	}
	return im
}

type imOperator struct {
	aiClient          ai.Interface
	userGetter        resources.Interface
	loginRecordGetter resources.Interface
	options           *authoptions.AuthenticationOptions
}

// UpdateUser returns user information after update.
func (im *imOperator) UpdateUser(new *iamv1.User) (*iamv1.User, error) {
	old, err := im.fetch(new.Name)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	if old.Annotations == nil {
		old.Annotations = make(map[string]string, 0)
	}
	// keep encrypted password
	new.Spec.EncryptedPassword = old.Spec.EncryptedPassword
	updated, err := im.aiClient.IamV1().Users().Update(context.Background(), old, metav1.UpdateOptions{})
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return ensurePasswordNotOutput(updated), nil
}

func (im *imOperator) fetch(username string) (*iamv1.User, error) {
	obj, err := im.userGetter.Get("", username)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	user := obj.(*iamv1.User).DeepCopy()
	return user, nil
}

func (im *imOperator) ModifyPassword(username string, password string) error {
	user, err := im.fetch(username)
	if err != nil {
		klog.Error(err)
		return err
	}
	user.Spec.EncryptedPassword = password
	_, err = im.aiClient.IamV1().Users().Update(context.Background(), user, metav1.UpdateOptions{})
	if err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

func (im *imOperator) ListUsers(query *query.Query) (list *iamv1.UserList, err error) {
	result, err := im.userGetter.List("", query)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	list = &iamv1.UserList{
		Items: make([]iamv1.User, 0),
	}
	for _, item := range result.Items {
		user := item.(*iamv1.User)
		out := ensurePasswordNotOutput(user)
		list.Items = append(list.Items, *out)
	}
	return list, nil
}

func (im *imOperator) PasswordVerify(username string, password string) error {
	obj, err := im.userGetter.Get("", username)
	if err != nil {
		klog.Error(err)
		return err
	}
	user := obj.(*iamv1.User)
	if err = auth.PasswordVerify(user.Spec.EncryptedPassword, password); err != nil {
		return err
	}
	return nil
}

func (im *imOperator) DescribeUser(username string) (*iamv1.User, error) {
	obj, err := im.userGetter.Get("", username)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	user := obj.(*iamv1.User)
	return ensurePasswordNotOutput(user), nil
}

func (im *imOperator) DeleteUser(username string) error {
	return im.aiClient.IamV1().Users().Delete(context.Background(), username, *metav1.NewDeleteOptions(0))
}

func (im *imOperator) CreateUser(user *iamv1.User) (*iamv1.User, error) {
	user, err := im.aiClient.IamV1().Users().Create(context.Background(), user, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return user, nil
}

func (im *imOperator) ListLoginRecords(username string, q *query.Query) (*iamv1.LoginRecordList, error) {
	q.Filters[query.FieldLabel] = query.Value(fmt.Sprintf("%s=%s", iamv1.UserReferenceLabel, username))
	result, err := im.loginRecordGetter.List("", q)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	list := &iamv1.LoginRecordList{
		Items: make([]iamv1.LoginRecord, 0),
	}
	for _, item := range result.Items {
		loginRecord := item.(*iamv1.LoginRecord)
		list.Items = append(list.Items, *loginRecord)
	}
	return list, nil
}

func ensurePasswordNotOutput(user *iamv1.User) *iamv1.User {
	out := user.DeepCopy()
	// ensure encrypted password will not be output
	out.Spec.EncryptedPassword = ""
	return out
}
