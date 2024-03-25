package loginrecord

import (
	"context"
	"errors"
	"time"

	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LoginRecordController struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *LoginRecordController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("loginrecord", req.NamespacedName)
	loginRecord := new(iamv1.LoginRecord)
	var err error
	if err = r.Get(ctx, req.NamespacedName, loginRecord); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("loginrecord is not exists", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch loginrecord")
		return ctrl.Result{}, err
	}

	if !loginRecord.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if err = r.updateUserLastLoginTime(loginRecord); err != nil {
		log.Error(err, "updateUserLastLoginTime loginRecord:"+req.Name)
		return ctrl.Result{}, err
	}

	now := time.Now()
	if loginRecord.CreationTimestamp.Add(time.Hour * 168).Before(now) {
		if err = r.Delete(context.Background(), loginRecord); err != nil {
			log.Error(err, "delete loginRecord:"+req.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LoginRecordController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iamv1.LoginRecord{}).
		Complete(r)
}

// updateUserLastLoginTime accepts a login object and set user lastLoginTime field
func (r *LoginRecordController) updateUserLastLoginTime(loginRecord *iamv1.LoginRecord) error {
	username, ok := loginRecord.Labels[iamv1.UserReferenceLabel]
	if !ok || len(username) == 0 {
		return errors.New("login doesn't belong to any user")
	}
	user := new(iamv1.User)
	var err error
	if err = r.Get(context.Background(), types.NamespacedName{Name: username}, user); err != nil {
		return err
	}

	// update lastLoginTime
	if user.DeletionTimestamp.IsZero() &&
		(user.Status.LastLoginTime == nil || user.Status.LastLoginTime.Before(&loginRecord.CreationTimestamp)) {
		user.Status.LastLoginTime = &loginRecord.CreationTimestamp
		return r.Status().Update(context.Background(), user)
	}
	return nil
}
