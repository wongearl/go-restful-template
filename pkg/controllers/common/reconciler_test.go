package common_test

import (
	"errors"
	"testing"

	"github.com/wongearl/go-restful-template/pkg/controllers/common"

	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestFeaturedReconcilerGroup(t *testing.T) {
	group := common.NewFeaturedReconcilerGroup()
	group.Put(&common.FakeReconciler{
		Group: "fake", Gate: "alpha",
	})
	group.Put(&common.FakeReconciler{
		Group: "fake", Gate: "beta", SetupError: errors.New("fake"),
	})
	group.Put(&common.FakeReconciler{
		Group: "fake", Gate: "release", SetupError: errors.New("fake"),
	})
	group.Put(&common.FakeReconciler{
		Group: "fake", Gate: "Alpha",
	})
	group.Put(&common.FakeReconciler{
		Group: "fake", Gate: "fake",
	})
	group.Put(&common.FakeReconciler{
		Group: "foo", Gate: "Alpha",
	})
	err := group.Init([]string{"fake"}, "alpha", nil)
	assert.NotNil(t, err)

	err = group.Init([]string{"fake", "foo"}, "fake", nil)
	assert.Nil(t, err)

	fakeReconciler := &common.FakeReconciler{}
	_, err = fakeReconciler.Reconcile(nil, ctrl.Request{})
	assert.Nil(t, err)
}
