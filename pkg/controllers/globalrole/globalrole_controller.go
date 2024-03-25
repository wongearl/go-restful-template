package globalrole

import (
	"context"

	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GlobalRoleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *GlobalRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("globalrole", req.NamespacedName)
	globalRole := new(iamv1.GlobalRole)
	var err error
	if err = r.Get(ctx, req.NamespacedName, globalRole); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("globalrole is not exists", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch globalrole")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil

}

func (r *GlobalRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iamv1.GlobalRole{}).
		Complete(r)
}
