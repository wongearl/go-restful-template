package user

import (
	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	alinformers "github.com/wongearl/go-restful-template/pkg/client/ai/informers/externalversions"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/klog"
)

type usersGetter struct {
	alInformer  alinformers.SharedInformerFactory
	k8sInformer k8sinformers.SharedInformerFactory
}

func New(alinformer alinformers.SharedInformerFactory, k8sinformer k8sinformers.SharedInformerFactory) v1alpha3.Interface {
	return &usersGetter{alInformer: alinformer, k8sInformer: k8sinformer}
}

func (d *usersGetter) Get(_, name string) (runtime.Object, error) {
	return d.alInformer.Iam().V1().Users().Lister().Get(name)
}

func (d *usersGetter) List(_ string, query *query.Query) (*api.ListResult, error) {
	var users []*iamv1.User
	var err error

	if namespace := query.Filters[iamv1.ScopeNamespace]; namespace != "" {
		role := query.Filters[iamv1.ResourcesSingularRole]
		users, err = d.listAllUsersInNamespace(string(namespace), string(role))
		delete(query.Filters, iamv1.ScopeNamespace)
		delete(query.Filters, iamv1.ResourcesSingularRole)
	} else if cluster := query.Filters[iamv1.ScopeCluster]; cluster == "true" {
		clusterRole := query.Filters[iamv1.ResourcesSingularClusterRole]
		users, err = d.listAllUsersInCluster(string(clusterRole))
		delete(query.Filters, iamv1.ScopeCluster)
		delete(query.Filters, iamv1.ResourcesSingularClusterRole)
	} else if globalRole := query.Filters[iamv1.ResourcesSingularGlobalRole]; globalRole != "" {
		users, err = d.listAllUsersByGlobalRole(string(globalRole))
		delete(query.Filters, iamv1.ResourcesSingularGlobalRole)
	} else {
		users, err = d.alInformer.Iam().V1().Users().Lister().List(query.Selector())
	}

	if err != nil {
		return nil, err
	}

	return v1alpha3.DefaultObjectList(users, query, d.compare, d.filter), nil
}

func (d *usersGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftUser, ok := left.(*iamv1.User)
	if !ok {
		return false
	}

	rightUser, ok := right.(*iamv1.User)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftUser.ObjectMeta, rightUser.ObjectMeta, field)
}

func (d *usersGetter) filter(object runtime.Object, filter query.Filter) bool {
	user, ok := object.(*iamv1.User)
	if !ok {
		return false
	}

	switch filter.Field {
	case iamv1.FieldEmail:
		return user.Spec.Email == string(filter.Value)
	default:
		return v1alpha3.DefaultObjectMetaFilter(user.ObjectMeta, filter)
	}
}

func (d *usersGetter) listAllUsersInNamespace(namespace, role string) ([]*iamv1.User, error) {
	var users []*iamv1.User
	var err error

	roleBindings, err := d.k8sInformer.Rbac().V1().
		RoleBindings().Lister().RoleBindings(namespace).List(labels.Everything())

	if err != nil {
		klog.Error(err)
		return nil, err
	}

	for _, roleBinding := range roleBindings {
		if role != "" && roleBinding.RoleRef.Name != role {
			continue
		}
		for _, subject := range roleBinding.Subjects {
			if subject.Kind == iamv1.ResourceKindUser {
				if contains(users, subject.Name) {
					klog.Warningf("conflict role binding found: %s, username:%s", roleBinding.ObjectMeta.String(), subject.Name)
					continue
				}

				obj, err := d.Get("", subject.Name)

				if err != nil {
					if errors.IsNotFound(err) {
						klog.Warningf("orphan subject: %s", subject.String())
						continue
					}
					klog.Error(err)
					return nil, err
				}

				user := obj.(*iamv1.User)
				user = user.DeepCopy()
				if user.Annotations == nil {
					user.Annotations = make(map[string]string, 0)
				}
				user.Annotations[iamv1.RoleAnnotation] = roleBinding.RoleRef.Name
				users = append(users, user)
			}
		}
	}

	return users, nil
}

func (d *usersGetter) listAllUsersByGlobalRole(globalRole string) ([]*iamv1.User, error) {
	var users []*iamv1.User
	var err error

	globalRoleBindings, err := d.alInformer.Iam().V1().
		GlobalRoleBindings().Lister().List(labels.Everything())

	if err != nil {
		klog.Error(err)
		return nil, err
	}

	for _, roleBinding := range globalRoleBindings {
		if roleBinding.RoleRef.Name != globalRole {
			continue
		}
		for _, subject := range roleBinding.Subjects {
			if subject.Kind == iamv1.ResourceKindUser {

				if contains(users, subject.Name) {
					klog.Warningf("conflict role binding found: %s, username:%s", roleBinding.ObjectMeta.String(), subject.Name)
					continue
				}

				obj, err := d.Get("", subject.Name)

				if err != nil {
					if errors.IsNotFound(err) {
						klog.Warningf("orphan subject: %s", subject.String())
						continue
					}
					klog.Error(err)
					return nil, err
				}

				user := obj.(*iamv1.User)
				user = user.DeepCopy()
				if user.Annotations == nil {
					user.Annotations = make(map[string]string, 0)
				}
				user.Annotations[iamv1.GlobalRoleAnnotation] = roleBinding.RoleRef.Name
				users = append(users, user)
			}
		}
	}

	return users, nil
}

func (d *usersGetter) listAllUsersInCluster(clusterRole string) ([]*iamv1.User, error) {
	var users []*iamv1.User
	var err error

	roleBindings, err := d.k8sInformer.Rbac().V1().ClusterRoleBindings().Lister().List(labels.Everything())

	if err != nil {
		klog.Error(err)
		return nil, err
	}

	for _, roleBinding := range roleBindings {
		if clusterRole != "" && roleBinding.RoleRef.Name != clusterRole {
			continue
		}
		for _, subject := range roleBinding.Subjects {
			if subject.Kind == iamv1.ResourceKindUser {
				if contains(users, subject.Name) {
					klog.Warningf("conflict role binding found: %s, username:%s", roleBinding.ObjectMeta.String(), subject.Name)
					continue
				}

				obj, err := d.Get("", subject.Name)

				if err != nil {
					if errors.IsNotFound(err) {
						klog.Warningf("orphan subject: %s", subject.String())
						continue
					}
					klog.Error(err)
					return nil, err
				}

				user := obj.(*iamv1.User)
				user = user.DeepCopy()
				if user.Annotations == nil {
					user.Annotations = make(map[string]string, 0)
				}
				user.Annotations[iamv1.ClusterRoleAnnotation] = roleBinding.RoleRef.Name
				users = append(users, user)
			}
		}
	}

	return users, nil
}

func contains(users []*iamv1.User, username string) bool {
	for _, user := range users {
		if user.Name == username {
			return true
		}
	}
	return false
}
