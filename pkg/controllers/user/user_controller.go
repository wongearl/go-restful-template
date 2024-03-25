package user

import (
	"context"
	"fmt"
	"time"

	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	"github.com/wongearl/go-restful-template/pkg/utils/sliceutil"

	"github.com/go-logr/logr"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// user finalizer
	finalizer = "finalizers.ai.io/users"
)

type UserReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("user", req.NamespacedName)
	user := new(iamv1.User)
	var err error
	if err = r.Get(ctx, req.NamespacedName, user); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("user is not exists", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch user")
		return ctrl.Result{}, err
	}

	if user.ObjectMeta.DeletionTimestamp.IsZero() {
		if !sliceutil.HasString(user.Finalizers, finalizer) {
			user.ObjectMeta.Finalizers = append(user.ObjectMeta.Finalizers, finalizer)
			if err = r.Update(ctx, user); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if sliceutil.HasString(user.ObjectMeta.Finalizers, finalizer) {
			log.V(1).Info("delete user", "name", req.Name)

			if err = r.deleteRoleBindings(user); err != nil {
				log.Error(err, "deleteRoleBindings user:"+req.Name)
				return ctrl.Result{}, err
			}

			if err = r.deleteLoginRecords(user); err != nil {
				log.Error(err, "deleteLoginRecords user:"+req.Name)
				return ctrl.Result{}, err
			}

			user.Finalizers = sliceutil.RemoveString(user.ObjectMeta.Finalizers, func(item string) bool {
				return item == finalizer
			})

			if err = r.Update(ctx, user); err != nil {
				return ctrl.Result{}, err
			}

		}
		return ctrl.Result{}, nil
	}

	if err = r.encryptPassword(user); err != nil {
		log.Error(err, "encryptPassword user:"+req.Name)
		return ctrl.Result{}, err
	}

	if err = r.syncUserStatus(user); err != nil {
		log.Error(err, "syncUserStatus user:"+req.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iamv1.User{}).
		Complete(r)
}

func (r *UserReconciler) deleteRoleBindings(user *iamv1.User) error {
	var err error

	var globalRoleBinding iamv1.GlobalRoleBinding
	if err = r.DeleteAllOf(context.Background(), &globalRoleBinding, client.MatchingLabels{iamv1.UserReferenceLabel: user.Name}); err != nil {
		return err
	}

	var namespaces corev1.NamespaceList
	if err = r.List(context.Background(), &namespaces); err != nil {
		return err
	}
	for _, namespace := range namespaces.Items {
		rolebinding := new(rbacv1.RoleBinding)
		if err = r.DeleteAllOf(context.Background(), rolebinding, client.InNamespace(namespace.Name), client.MatchingLabels{iamv1.UserReferenceLabel: user.Name}); err != nil {
			return err
		}
	}

	return nil
}

func (r *UserReconciler) deleteLoginRecords(user *iamv1.User) error {
	var record iamv1.LoginRecord
	if err := r.DeleteAllOf(context.Background(), &record, client.MatchingLabels{iamv1.UserReferenceLabel: user.Name}); err != nil {
		return err
	}
	return nil
}

func (r *UserReconciler) encryptPassword(user *iamv1.User) error {
	if user.Spec.EncryptedPassword != "" && !isEncrypted(user.Spec.EncryptedPassword) {
		password, err := encrypt(user.Spec.EncryptedPassword)
		if err != nil {
			return err
		}
		user = user.DeepCopy()
		user.Spec.EncryptedPassword = password
		if user.Annotations == nil {
			user.Annotations = make(map[string]string)
		}
		// ensure plain text password won't be kept anywhere
		delete(user.Annotations, corev1.LastAppliedConfigAnnotation)
		return r.Update(context.Background(), user)
	}
	return nil
}

func (r *UserReconciler) syncUserStatus(user *iamv1.User) error {
	if user.Status.State != nil && *user.Status.State == iamv1.UserDisabled {
		return nil
	}

	if isEncrypted(user.Spec.EncryptedPassword) &&
		user.Status.State == nil {
		expected := user.DeepCopy()
		active := iamv1.UserActive
		expected.Status = iamv1.UserStatus{
			State:              &active,
			LastTransitionTime: &metav1.Time{Time: time.Now()},
		}
		return r.Status().Update(context.Background(), expected)
	}

	if user.Status.State != nil && *user.Status.State == iamv1.UserAuthLimitExceeded {
		if user.Status.LastTransitionTime != nil &&
			user.Status.LastTransitionTime.Add(time.Minute*10).Before(time.Now()) {
			expected := user.DeepCopy()
			// unblock user
			active := iamv1.UserActive
			expected.Status = iamv1.UserStatus{
				State:              &active,
				LastTransitionTime: &metav1.Time{Time: time.Now()},
			}
			return r.Status().Update(context.Background(), expected)
		}
	}

	var records iamv1.LoginRecordList

	err := r.List(context.Background(), &records, client.MatchingLabels{iamv1.UserReferenceLabel: user.Name})
	if err != nil {
		return err
	}

	now := time.Now()
	failedLoginAttempts := 0
	for _, loginRecord := range records.Items {
		if !loginRecord.Spec.Success &&
			loginRecord.CreationTimestamp.Add(time.Minute*10).After(now) {
			failedLoginAttempts++
		}
	}

	if failedLoginAttempts >= 10 {
		expect := user.DeepCopy()
		limitExceed := iamv1.UserAuthLimitExceeded
		expect.Status = iamv1.UserStatus{
			State:              &limitExceed,
			Reason:             fmt.Sprintf("Failed login attempts exceed %d in last 10min", failedLoginAttempts),
			LastTransitionTime: &metav1.Time{Time: time.Now()},
		}
		return r.Status().Update(context.Background(), expect)
	}
	return nil

}

func encrypt(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func isEncrypted(password string) bool {
	// bcrypt.Cost returns the hashing cost used to create the given hashed
	cost, _ := bcrypt.Cost([]byte(password))
	// cost > 0 means the password has been encrypted
	return cost > 0
}
