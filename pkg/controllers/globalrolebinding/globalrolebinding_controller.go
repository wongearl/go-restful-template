package globalrolebinding

import (
	"context"
	"fmt"

	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type GlobalRoleBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *GlobalRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("globalrolebinding", req.NamespacedName)
	globalRoleBinding := new(iamv1.GlobalRoleBinding)
	var err error
	if err = r.Get(ctx, req.NamespacedName, globalRoleBinding); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("globalrolebinding is not exists", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch globalrolebinding")
		return ctrl.Result{}, err
	}
	if globalRoleBinding.RoleRef.Name == iamv1.PlatformAdmin {
		if err = r.assignClusterAdminRole(globalRoleBinding); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil

}

func (r *GlobalRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iamv1.GlobalRoleBinding{}).
		Complete(r)
}

func (r *GlobalRoleBindingReconciler) assignClusterAdminRole(globalRoleBinding *iamv1.GlobalRoleBinding) error {

	username := findExpectUsername(globalRoleBinding)
	if username == "" {
		return nil
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", username, iamv1.ClusterAdmin),
		},
		Subjects: ensureSubjectAPIVersionIsValid(globalRoleBinding.Subjects),
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     iamv1.ResourceKindClusterRole,
			Name:     iamv1.ClusterAdmin,
		},
	}

	err := controllerutil.SetControllerReference(globalRoleBinding, clusterRoleBinding, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Create(context.Background(), clusterRoleBinding)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}

func findExpectUsername(globalRoleBinding *iamv1.GlobalRoleBinding) string {
	for _, subject := range globalRoleBinding.Subjects {
		if subject.Kind == iamv1.ResourceKindUser {
			return subject.Name
		}
	}
	return ""
}

func ensureSubjectAPIVersionIsValid(subjects []rbacv1.Subject) []rbacv1.Subject {
	validSubjects := make([]rbacv1.Subject, 0)
	for _, subject := range subjects {
		if subject.Kind == iamv1.ResourceKindUser {
			validSubject := rbacv1.Subject{
				Kind:     iamv1.ResourceKindUser,
				APIGroup: "rbac.authorization.k8s.io",
				Name:     subject.Name,
			}
			validSubjects = append(validSubjects, validSubject)
		}
	}
	return validSubjects
}
