package core

import (
	"context"
	"errors"

	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// FakeGroupedReconciler is a fake GroupedReconciler which is for the test purpose
type FakeGroupedReconciler struct {
	HasError bool
}

// GetName returns the name
func (f *FakeGroupedReconciler) GetName() string {
	return "fake"
}

// GetGroupName returns the group name
func (f *FakeGroupedReconciler) GetGroupName() string {
	return "fake"
}

// Reconcile is fake reconcile process
func (f *FakeGroupedReconciler) Reconcile(context.Context, reconcile.Request) (result reconcile.Result, err error) {
	return
}

// SetupWithManager setups the reconciler
func (f *FakeGroupedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if f.HasError {
		return errors.New("fake")
	}
	return nil
}

// NoErrors represents no errors
func NoErrors(t assert.TestingT, err error, i ...interface{}) bool {
	assert.Nil(t, err)
	return true
}
