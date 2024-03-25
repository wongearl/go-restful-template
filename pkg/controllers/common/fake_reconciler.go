package common

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

// FakeReconciler only for the test purpose
type FakeReconciler struct {
	SetupError error
	Group      string
	Name       string
	Gate       string
}

func (f *FakeReconciler) Reconcile(context.Context, ctrl.Request) (res ctrl.Result, err error) {
	return
}

func (f *FakeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return f.SetupError
}

func (f *FakeReconciler) FeatureGroup() string {
	return f.Group
}

func (f *FakeReconciler) FeatureName() string {
	return f.Name
}

func (f *FakeReconciler) FeatureGate() string {
	return f.Gate
}
