package common

import (
	"context"
	"fmt"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
)

// Reconciler represents a reconciler interface
type Reconciler interface {
	Reconcile(context.Context, ctrl.Request) (ctrl.Result, error)
	SetupWithManager(mgr ctrl.Manager) error
}

// FeaturedReconciler represents a featured reconciler
type FeaturedReconciler interface {
	Reconciler
	FeatureGroup() string
	FeatureName() string
	FeatureGate() string
}

// FeaturedReconcilerGroup groups a set of reconcilers
type FeaturedReconcilerGroup struct {
	controllers featuredReconcilers
}

var gateMap = map[string]int{
	"release": 1,
	"beta":    2,
	"alpha":   3,
}

func getGateValue(name string) int {
	if val, ok := gateMap[strings.ToLower(name)]; !ok {
		return -1
	} else {
		return val
	}
}

type featuredReconcilers []FeaturedReconciler

func (f featuredReconcilers) init(name, featureGate string, mgr ctrl.Manager) (err error) {
	for _, reconciler := range f {
		if reconciler.FeatureGroup() != name {
			continue
		}
		if getGateValue(featureGate) == -1 ||
			getGateValue(reconciler.FeatureGate()) == -1 ||
			getGateValue(featureGate) < getGateValue(reconciler.FeatureGate()) {
			continue
		}

		if err = reconciler.SetupWithManager(mgr); err != nil {
			break
		}
		fmt.Printf("controller %s-%s-%s has started\n",
			reconciler.FeatureGroup(), reconciler.FeatureName(), reconciler.FeatureGate())
	}
	return
}

// NewFeaturedReconcilerGroup creates the FeaturedReconcilerGroup instance
func NewFeaturedReconcilerGroup() *FeaturedReconcilerGroup {
	return &FeaturedReconcilerGroup{
		controllers: featuredReconcilers{},
	}
}

// Put puts a reconciler
func (g *FeaturedReconcilerGroup) Put(reconciler FeaturedReconciler) {
	g.controllers = append(g.controllers, reconciler)
}

// Init setups all the controllers
func (g *FeaturedReconcilerGroup) Init(featureSlice []string, featureGate string, mgr ctrl.Manager) (err error) {
	for _, feature := range featureSlice {
		if err = g.controllers.init(feature, featureGate, mgr); err != nil {
			break
		}
	}
	return
}
