package am

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"

	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	ai "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned"
	"github.com/wongearl/go-restful-template/pkg/client/informers"
	"github.com/wongearl/go-restful-template/pkg/constants"
	resourcev1alpha3 "github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/clusterrole"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/clusterrolebinding"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/globalrole"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/globalrolebinding"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/role"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/rolebinding"
	"github.com/wongearl/go-restful-template/pkg/utils/sliceutil"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

type AccessManagementInterface interface {
	GetGlobalRoleOfUser(username string) (*iamv1.GlobalRole, error)
	GetClusterRoleOfUser(username string) (*rbacv1.ClusterRole, error)
	GetNamespaceRoleOfUser(username string, groups []string, namespace string) ([]*rbacv1.Role, error)
	ListRoles(namespace string, query *query.Query) (*rbacv1.RoleList, error)
	ListClusterRoles(query *query.Query) (*api.ListResult, error)
	ListGlobalRoles(query *query.Query) (*iamv1.GlobalRoleList, error)
	ListGlobalRoleBindings(username string) ([]*iamv1.GlobalRoleBinding, error)
	ListClusterRoleBindings(username string) ([]*rbacv1.ClusterRoleBinding, error)
	ListRoleBindings(username string, groups []string, namespace string) ([]*rbacv1.RoleBinding, error)
	GetRoleBindingOfUser(username string) ([]*rbacv1.RoleBinding, error)
	GetRoleReferenceRules(roleRef rbacv1.RoleRef, namespace string) (string, []rbacv1.PolicyRule, error)
	GetGlobalRole(globalRole string) (*iamv1.GlobalRole, error)
	CreateGlobalRoleBinding(username string, globalRole string) error
	CreateOrUpdateGlobalRole(globalRole *iamv1.GlobalRole) (*iamv1.GlobalRole, error)
	PatchGlobalRole(globalRole *iamv1.GlobalRole) (*iamv1.GlobalRole, error)
	DeleteGlobalRole(name string) error
	CreateOrUpdateClusterRole(clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error)
	DeleteClusterRole(name string) error
	GetClusterRole(name string) (*rbacv1.ClusterRole, error)
	GetNamespaceRole(namespace string, name string) (*rbacv1.Role, error)
	CreateOrUpdateNamespaceRole(namespace string, role *rbacv1.Role) (*rbacv1.Role, error)
	DeleteNamespaceRole(namespace string, name string) error
	CreateNamespaceRoleBinding(username string, namespace string, role string) error
	RemoveUserFromNamespace(username string, namespace string) error
	CreateClusterRoleBinding(username string, role string) error
	RemoveUserFromCluster(username string) error
	PatchNamespaceRole(namespace string, role *rbacv1.Role) (*rbacv1.Role, error)
	PatchClusterRole(clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error)
	CreateRoleBinding(namespace string, roleBinding *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)
	DeleteRoleBinding(namespace, name string) error
	GetNamespaceRoleBindingByUser(namespace, username string) *rbacv1.RoleBinding
	DeleteNamespaceRoleBindingByUser(namespace, username string) error
}

type amOperator struct {
	globalRoleBindingGetter  resourcev1alpha3.Interface
	clusterRoleBindingGetter resourcev1alpha3.Interface
	roleBindingGetter        resourcev1alpha3.Interface
	globalRoleGetter         resourcev1alpha3.Interface
	clusterRoleGetter        resourcev1alpha3.Interface
	roleGetter               resourcev1alpha3.Interface
	namespaceLister          listersv1.NamespaceLister
	aiclient                 ai.Interface
	k8sclient                kubernetes.Interface
}

func NewReadOnlyOperator(factory informers.InformerFactory) AccessManagementInterface {
	return &amOperator{
		globalRoleBindingGetter: globalrolebinding.New(factory.AiSharedInformerFactory()),

		clusterRoleBindingGetter: clusterrolebinding.New(factory.KubernetesSharedInformerFactory()),
		roleBindingGetter:        rolebinding.New(factory.KubernetesSharedInformerFactory()),
		globalRoleGetter:         globalrole.New(factory.AiSharedInformerFactory()),
		clusterRoleGetter:        clusterrole.New(factory.KubernetesSharedInformerFactory()),
		roleGetter:               role.New(factory.KubernetesSharedInformerFactory()),
		namespaceLister:          factory.KubernetesSharedInformerFactory().Core().V1().Namespaces().Lister(),
	}
}

func NewOperator(aiClient ai.Interface, k8sClient kubernetes.Interface, factory informers.InformerFactory) AccessManagementInterface {
	amOperator := NewReadOnlyOperator(factory).(*amOperator)
	amOperator.aiclient = aiClient
	amOperator.k8sclient = k8sClient
	return amOperator
}

func (am *amOperator) GetGlobalRoleOfUser(username string) (*iamv1.GlobalRole, error) {
	globalRoleBindings, err := am.ListGlobalRoleBindings(username)
	if len(globalRoleBindings) > 0 {
		// Usually, only one globalRoleBinding will be found which is created from ks-console.
		if len(globalRoleBindings) > 1 {
			klog.Warningf("conflict global role binding, username: %s", username)
		}
		globalRole, err := am.GetGlobalRole(globalRoleBindings[0].RoleRef.Name)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
		return globalRole, nil
	}

	err = errors.NewNotFound(iamv1.Resource(iamv1.ResourcesSingularGlobalRoleBinding), username)
	klog.V(4).Info(err)
	return nil, err
}

func (am *amOperator) GetNamespaceRoleOfUser(username string, groups []string, namespace string) ([]*rbacv1.Role, error) {

	userRoleBindings, err := am.ListRoleBindings(username, groups, namespace)

	if err != nil {
		klog.Error(err)
		return nil, err
	}

	if len(userRoleBindings) > 0 {
		roles := make([]*rbacv1.Role, len(userRoleBindings))
		for i, roleBinding := range userRoleBindings {
			role, err := am.GetNamespaceRole(namespace, roleBinding.RoleRef.Name)
			if err != nil {
				klog.Error(err)
				return nil, err
			}

			out := role.DeepCopy()
			if out.Annotations == nil {
				out.Annotations = make(map[string]string, 0)
			}
			out.Annotations[iamv1.RoleAnnotation] = role.Name

			roles[i] = out
		}

		if len(userRoleBindings) > 1 {
			klog.Infof("conflict role binding, username: %s", username)
		}
		return roles, nil
	}

	err = errors.NewNotFound(iamv1.Resource(iamv1.ResourcesSingularRoleBinding), username)
	klog.V(4).Info(err)
	return nil, err
}

func (am *amOperator) GetClusterRoleOfUser(username string) (*rbacv1.ClusterRole, error) {
	userRoleBindings, err := am.ListClusterRoleBindings(username)

	if err != nil {
		klog.Error(err)
		return nil, err
	}

	if len(userRoleBindings) > 0 {
		role, err := am.GetClusterRole(userRoleBindings[0].RoleRef.Name)
		if err != nil {
			klog.Error(err)
			return nil, err
		}

		if len(userRoleBindings) > 1 {
			klog.Warningf("conflict cluster role binding, username: %s", username)
		}

		out := role.DeepCopy()
		if out.Annotations == nil {
			out.Annotations = make(map[string]string, 0)
		}
		out.Annotations[iamv1.ClusterRoleAnnotation] = role.Name
		return out, nil
	}

	err = errors.NewNotFound(iamv1.Resource(iamv1.ResourcesSingularClusterRoleBinding), username)
	klog.V(4).Info(err)
	return nil, err
}

func (am *amOperator) ListClusterRoleBindings(username string) ([]*rbacv1.ClusterRoleBinding, error) {

	roleBindings, err := am.clusterRoleBindingGetter.List("", query.New())
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.ClusterRoleBinding, 0)
	for _, obj := range roleBindings.Items {
		roleBinding := obj.(*rbacv1.ClusterRoleBinding)
		if contains(roleBinding.Subjects, username, nil) {
			result = append(result, roleBinding)
		}
	}

	return result, nil
}

func (am *amOperator) ListGlobalRoleBindings(username string) ([]*iamv1.GlobalRoleBinding, error) {
	roleBindings, err := am.globalRoleBindingGetter.List("", query.New())
	if err != nil {
		return nil, err
	}

	result := make([]*iamv1.GlobalRoleBinding, 0)
	for _, obj := range roleBindings.Items {
		roleBinding := obj.(*iamv1.GlobalRoleBinding)
		if contains(roleBinding.Subjects, username, nil) {
			result = append(result, roleBinding)
		}
	}

	return result, nil
}

func (am *amOperator) ListRoleBindings(username string, groups []string, namespace string) ([]*rbacv1.RoleBinding, error) {
	roleBindings, err := am.roleBindingGetter.List(namespace, query.New())
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	result := make([]*rbacv1.RoleBinding, 0)
	for _, obj := range roleBindings.Items {
		roleBinding := obj.(*rbacv1.RoleBinding)
		if contains(roleBinding.Subjects, username, groups) {
			result = append(result, roleBinding)
		}
	}
	return result, nil
}

func (am *amOperator) GetRoleBindingOfUser(username string) ([]*rbacv1.RoleBinding, error) {
	roleBindings, err := am.roleBindingGetter.List("", query.New())
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	result := make([]*rbacv1.RoleBinding, 0)
	for _, obj := range roleBindings.Items {
		roleBinding := obj.(*rbacv1.RoleBinding)
		if roleBinding.Labels[iamv1.UserReferenceLabel] == username {
			result = append(result, roleBinding)
		}
	}
	return result, nil
}

func contains(subjects []rbacv1.Subject, username string, groups []string) bool {
	// if username is nil means list all role bindings
	if username == "" {
		return true
	}
	for _, subject := range subjects {
		if subject.Kind == rbacv1.UserKind && subject.Name == username {
			return true
		}
		if subject.Kind == rbacv1.GroupKind && sliceutil.HasString(groups, subject.Name) {
			return true
		}
	}
	return false
}

func (am *amOperator) ListRoles(namespace string, query *query.Query) (*rbacv1.RoleList, error) {
	result, err := am.roleGetter.List(namespace, query)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	list := &rbacv1.RoleList{
		Items: make([]rbacv1.Role, 0),
	}
	for _, item := range result.Items {
		role := item.(*rbacv1.Role)
		list.Items = append(list.Items, *role)
	}
	return list, nil
}

func (am *amOperator) ListClusterRoles(query *query.Query) (*api.ListResult, error) {
	return am.clusterRoleGetter.List("", query)
}

func (am *amOperator) ListGlobalRoles(query *query.Query) (*iamv1.GlobalRoleList, error) {
	result, err := am.globalRoleGetter.List("", query)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	list := &iamv1.GlobalRoleList{
		Items: make([]iamv1.GlobalRole, 0),
	}
	for _, item := range result.Items {
		globalRole := item.(*iamv1.GlobalRole)
		list.Items = append(list.Items, *globalRole)
	}
	return list, nil
}

func (am *amOperator) GetGlobalRole(globalRole string) (*iamv1.GlobalRole, error) {
	obj, err := am.globalRoleGetter.Get("", globalRole)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return obj.(*iamv1.GlobalRole), nil
}

func (am *amOperator) CreateGlobalRoleBinding(username string, role string) error {
	_, err := am.GetGlobalRole(role)
	if err != nil {
		klog.Error(err)
		return err
	}

	roleBindings, err := am.ListGlobalRoleBindings(username)
	if err != nil {
		klog.Error(err)
		return err
	}

	for _, roleBinding := range roleBindings {
		if role == roleBinding.RoleRef.Name {
			return nil
		}
		err := am.aiclient.IamV1().GlobalRoleBindings().Delete(context.Background(), roleBinding.Name, *metav1.NewDeleteOptions(0))
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			klog.Error(err)
			return err
		}
	}

	globalRoleBinding := iamv1.GlobalRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", username, role),
			Labels: map[string]string{iamv1.UserReferenceLabel: username},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Name:     username,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: iamv1.SchemeGroupVersion.Group,
			Kind:     iamv1.ResourceKindGlobalRole,
			Name:     role,
		},
	}

	if _, err := am.aiclient.IamV1().GlobalRoleBindings().Create(context.Background(), &globalRoleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (am *amOperator) PatchGlobalRole(globalRole *iamv1.GlobalRole) (*iamv1.GlobalRole, error) {
	old, err := am.GetGlobalRole(globalRole.Name)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	// rules cannot be override
	globalRole.Rules = old.Rules

	data, err := json.Marshal(globalRole)
	if err != nil {
		return nil, err
	}

	return am.aiclient.IamV1().GlobalRoles().Patch(context.Background(), globalRole.Name, types.MergePatchType, data, metav1.PatchOptions{})
}

func (am *amOperator) PatchNamespaceRole(namespace string, role *rbacv1.Role) (*rbacv1.Role, error) {
	old, err := am.GetNamespaceRole(namespace, role.Name)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	// rules cannot be override
	role.Rules = old.Rules

	data, err := json.Marshal(role)
	if err != nil {
		return nil, err
	}

	return am.k8sclient.RbacV1().Roles(namespace).Patch(context.Background(), role.Name, types.MergePatchType, data, metav1.PatchOptions{})
}

func (am *amOperator) PatchClusterRole(clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	old, err := am.GetClusterRole(clusterRole.Name)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	// rules cannot be override
	clusterRole.Rules = old.Rules

	data, err := json.Marshal(clusterRole)
	if err != nil {
		return nil, err
	}

	return am.k8sclient.RbacV1().ClusterRoles().Patch(context.Background(), clusterRole.Name, types.MergePatchType, data, metav1.PatchOptions{})
}

func (am *amOperator) CreateClusterRoleBinding(username string, role string) error {
	_, err := am.GetClusterRole(role)
	if err != nil {
		klog.Error(err)
		return err
	}

	roleBindings, err := am.ListClusterRoleBindings(username)

	if err != nil {
		klog.Error(err)
		return err
	}

	for _, roleBinding := range roleBindings {
		if role == roleBinding.RoleRef.Name {
			return nil
		}
		err := am.k8sclient.RbacV1().ClusterRoleBindings().Delete(context.Background(), roleBinding.Name, *metav1.NewDeleteOptions(0))
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			klog.Error(err)
			return err
		}
	}

	roleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", username, role),
			Labels: map[string]string{iamv1.UserReferenceLabel: username},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Name:     username,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     iamv1.ResourceKindClusterRole,
			Name:     role,
		},
	}

	if _, err := am.k8sclient.RbacV1().ClusterRoleBindings().Create(context.Background(), &roleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (am *amOperator) CreateNamespaceRoleBinding(username string, namespace string, role string) error {

	ns, err := am.k8sclient.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return err
	}
	manager := ns.Annotations[constants.CreatorAnnotationKey]
	if role != "admin" {
		if manager == username {
			return goerrors.New("需先指定新的项目管理员!")
		}
	}

	err = am.DeleteNamespaceRoleBindingByUser(namespace, username)
	if err != nil {
		klog.Error(err)
		return err
	}

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", username, role),
			Labels: map[string]string{iamv1.UserReferenceLabel: username},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Name:     username,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     iamv1.ResourceKindRole,
			Name:     role,
		},
	}

	if _, err := am.k8sclient.RbacV1().RoleBindings(namespace).Create(context.Background(), &roleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}

	if role == "admin" {
		if manager != username {
			err = am.DeleteNamespaceRoleBindingByUser(namespace, manager)
			if err != nil {
				klog.Error(err)
				return err
			}

			roleBinding = rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:   fmt.Sprintf("%s-operator", manager),
					Labels: map[string]string{iamv1.UserReferenceLabel: manager},
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:     rbacv1.UserKind,
						APIGroup: rbacv1.SchemeGroupVersion.Group,
						Name:     manager,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.SchemeGroupVersion.Group,
					Kind:     iamv1.ResourceKindRole,
					Name:     "operator",
				},
			}

			if _, err := am.k8sclient.RbacV1().RoleBindings(namespace).Create(context.Background(), &roleBinding, metav1.CreateOptions{}); err != nil {
				return err
			}

			ns.Annotations[constants.CreatorAnnotationKey] = username
			if _, err := am.k8sclient.CoreV1().Namespaces().Update(context.Background(), ns, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (am *amOperator) RemoveUserFromNamespace(username string, namespace string) error {
	ns, err := am.k8sclient.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return err
	}
	manager := ns.Annotations[constants.CreatorAnnotationKey]
	if manager == username {
		return goerrors.New("用户:" + username + "是项目:" + namespace + "的管理员，需更换该项目管理员才能移除此用户!")
	}

	err = am.DeleteNamespaceRoleBindingByUser(namespace, username)
	if err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

func (am *amOperator) RemoveUserFromCluster(username string) error {
	roleBindings, err := am.ListClusterRoleBindings(username)
	if err != nil {
		klog.Error(err)
		return err
	}

	for _, roleBinding := range roleBindings {
		err := am.k8sclient.RbacV1().ClusterRoleBindings().Delete(context.Background(), roleBinding.Name, *metav1.NewDeleteOptions(0))
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			klog.Error(err)
			return err
		}
	}

	return nil
}

func (am *amOperator) CreateOrUpdateGlobalRole(globalRole *iamv1.GlobalRole) (*iamv1.GlobalRole, error) {
	globalRole.Rules = make([]rbacv1.PolicyRule, 0)
	var created *iamv1.GlobalRole
	var err error
	if globalRole.ResourceVersion != "" {
		created, err = am.aiclient.IamV1().GlobalRoles().Update(context.Background(), globalRole, metav1.UpdateOptions{})
	} else {
		created, err = am.aiclient.IamV1().GlobalRoles().Create(context.Background(), globalRole, metav1.CreateOptions{})
	}
	return created, err
}

func (am *amOperator) CreateOrUpdateClusterRole(clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	clusterRole.Rules = make([]rbacv1.PolicyRule, 0)
	var created *rbacv1.ClusterRole
	var err error
	if clusterRole.ResourceVersion != "" {
		created, err = am.k8sclient.RbacV1().ClusterRoles().Update(context.Background(), clusterRole, metav1.UpdateOptions{})
	} else {
		created, err = am.k8sclient.RbacV1().ClusterRoles().Create(context.Background(), clusterRole, metav1.CreateOptions{})
	}
	return created, err
}

func (am *amOperator) CreateOrUpdateNamespaceRole(namespace string, role *rbacv1.Role) (*rbacv1.Role, error) {
	role.Rules = make([]rbacv1.PolicyRule, 0)
	role.Namespace = namespace
	var created *rbacv1.Role
	var err error
	if role.ResourceVersion != "" {
		created, err = am.k8sclient.RbacV1().Roles(namespace).Update(context.Background(), role, metav1.UpdateOptions{})
	} else {
		created, err = am.k8sclient.RbacV1().Roles(namespace).Create(context.Background(), role, metav1.CreateOptions{})
	}

	return created, err
}

func (am *amOperator) DeleteGlobalRole(name string) error {
	return am.aiclient.IamV1().GlobalRoles().Delete(context.Background(), name, *metav1.NewDeleteOptions(0))
}

func (am *amOperator) DeleteClusterRole(name string) error {
	return am.k8sclient.RbacV1().ClusterRoles().Delete(context.Background(), name, *metav1.NewDeleteOptions(0))
}
func (am *amOperator) DeleteNamespaceRole(namespace string, name string) error {
	return am.k8sclient.RbacV1().Roles(namespace).Delete(context.Background(), name, *metav1.NewDeleteOptions(0))
}

// GetRoleReferenceRules attempts to resolve the RoleBinding or ClusterRoleBinding.
func (am *amOperator) GetRoleReferenceRules(roleRef rbacv1.RoleRef, namespace string) (regoPolicy string, rules []rbacv1.PolicyRule, err error) {

	empty := make([]rbacv1.PolicyRule, 0)

	switch roleRef.Kind {
	case iamv1.ResourceKindRole:
		role, err := am.GetNamespaceRole(namespace, roleRef.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				return "", empty, nil
			}
			return "", nil, err
		}
		return role.Annotations[iamv1.RegoOverrideAnnotation], role.Rules, nil
	case iamv1.ResourceKindClusterRole:
		clusterRole, err := am.GetClusterRole(roleRef.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				return "", empty, nil
			}
			return "", nil, err
		}
		return clusterRole.Annotations[iamv1.RegoOverrideAnnotation], clusterRole.Rules, nil
	case iamv1.ResourceKindGlobalRole:
		globalRole, err := am.GetGlobalRole(roleRef.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				return "", empty, nil
			}
			return "", nil, err
		}
		return globalRole.Annotations[iamv1.RegoOverrideAnnotation], globalRole.Rules, nil

	default:
		return "", nil, fmt.Errorf("unsupported role reference kind: %q", roleRef.Kind)
	}
}

func (am *amOperator) GetNamespaceRole(namespace string, name string) (*rbacv1.Role, error) {
	obj, err := am.roleGetter.Get(namespace, name)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return obj.(*rbacv1.Role), nil
}

func (am *amOperator) GetClusterRole(name string) (*rbacv1.ClusterRole, error) {
	obj, err := am.clusterRoleGetter.Get("", name)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return obj.(*rbacv1.ClusterRole), nil
}

func (am *amOperator) GetNamespaceRoleBindingByUser(namespace, username string) *rbacv1.RoleBinding {
	roleBindingList, err := am.k8sclient.RbacV1().RoleBindings(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Error(err)
		return nil
	}

	for _, nsRolebinding := range roleBindingList.Items {
		rolebinding := nsRolebinding
		if rolebinding.Labels[iamv1.UserReferenceLabel] == username {
			return &rolebinding
		}
	}
	return nil
}

func (am *amOperator) DeleteNamespaceRoleBindingByUser(namespace, username string) error {
	nsRoleBinding := am.GetNamespaceRoleBindingByUser(namespace, username)
	if nsRoleBinding != nil {
		return am.k8sclient.RbacV1().RoleBindings(namespace).Delete(context.Background(), nsRoleBinding.Name, *metav1.NewDeleteOptions(0))
	}
	return nil
}

func (am *amOperator) CreateRoleBinding(namespace string, roleBinding *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {

	_, err := am.GetNamespaceRole(namespace, roleBinding.RoleRef.Name)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	if len(roleBinding.Subjects) == 0 {
		err := errors.NewNotFound(iamv1.Resource(iamv1.ResourcesPluralUser), "")
		return nil, err
	}

	roleBinding.GenerateName = fmt.Sprintf("%s-%s-", roleBinding.Subjects[0].Name, roleBinding.RoleRef.Name)

	if roleBinding.Labels == nil {
		roleBinding.Labels = map[string]string{}
	}

	roleBinding.Labels[iamv1.UserReferenceLabel] = roleBinding.Subjects[0].Name

	return am.k8sclient.RbacV1().RoleBindings(namespace).Create(context.Background(), roleBinding, metav1.CreateOptions{})
}

func (am *amOperator) DeleteRoleBinding(namespace, name string) error {
	return am.k8sclient.RbacV1().RoleBindings(namespace).Delete(context.Background(), name, *metav1.NewDeleteOptions(0))
}
