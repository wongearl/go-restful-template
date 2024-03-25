package core_test

import (
	"context"
	"errors"
	corev1 "github.com/wongearl/go-restful-template/pkg/api/core.ai.io/v1"
	"github.com/wongearl/go-restful-template/pkg/controllers/core"
	"reflect"
	"testing"
	"time"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRegularReconcile(t *testing.T) {
	tests := []struct {
		name           string
		health         *corev1.Health
		err            error
		expectedResult ctrl.Result
	}{{
		name:           "err is nil, without interval setting",
		health:         &corev1.Health{},
		expectedResult: ctrl.Result{RequeueAfter: time.Second * 30},
	}, {
		name:           "err is nil, with interval setting",
		health:         &corev1.Health{Spec: corev1.HealthSpec{Interval: time.Second}},
		expectedResult: ctrl.Result{RequeueAfter: time.Second},
	}, {
		name:           "err is not nil",
		health:         &corev1.Health{},
		err:            errors.New("test"),
		expectedResult: ctrl.Result{},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := core.RegularReconcile(tt.health, tt.err); !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("RegularReconcile() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestRepositoryReconcile(t *testing.T) {
	schema, err := corev1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	err = corev1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)
	err = v1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)

	defaultRequest := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "fake", Name: "fake"},
	}

	defaultHealth := &corev1.Health{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake",
			Namespace: "fake",
		},
		Spec: corev1.HealthSpec{
			ComponentSelectors: map[string]string{"fake": "app=fake"},
		},
	}
	withoutSelectors := defaultHealth.DeepCopy()
	withoutSelectors.Spec.ComponentSelectors = nil

	invalidSelectors := defaultHealth.DeepCopy()
	invalidSelectors.Spec.ComponentSelectors = map[string]string{"fake": "@"}

	healthWithHTTP := defaultHealth.DeepCopy()
	healthWithHTTP.Spec.ComponentSelectors = nil
	healthWithHTTP.Spec.ServiceProbes = map[string]v1.Probe{
		"fake": {
			ProbeHandler: v1.ProbeHandler{
				HTTPGet: &v1.HTTPGetAction{
					Path: "/healthz",
				},
			},
		},
	}

	defaultPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake",
			Namespace: "fake",
			Labels:    map[string]string{"app": "fake"},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{{
				RestartCount: 1,
			}},
		},
	}

	tests := []struct {
		name    string
		mgr     ctrl.Manager
		req     ctrl.Request
		verify  func(*testing.T, client.Client)
		prepare func()
		wantErr bool
	}{{
		name: "not found",
		mgr: &core.FakeManager{
			Client: fake.NewClientBuilder().WithScheme(schema).Build(),
			Scheme: schema,
		},
		req:     defaultRequest,
		wantErr: false,
	}, {
		name: "without any selectors",
		mgr: &core.FakeManager{
			Client: fake.NewClientBuilder().WithScheme(schema).WithObjects(withoutSelectors).Build(),
			Scheme: schema,
		},
		req: defaultRequest,
		verify: func(t *testing.T, c client.Client) {
			target := &corev1.Health{}
			err := c.Get(context.Background(), defaultRequest.NamespacedName, target)
			assert.NoError(t, err)
			assert.Equal(t, corev1.Status{}, target.Status)
		},
		wantErr: false,
	}, {
		name: "normal case",
		mgr: &core.FakeManager{
			Client: fake.NewClientBuilder().WithScheme(schema).WithObjects(defaultHealth, defaultPod).Build(),
			Scheme: schema,
		},
		req: defaultRequest,
		verify: func(t *testing.T, c client.Client) {
			target := &corev1.Health{}
			err := c.Get(context.Background(), defaultRequest.NamespacedName, target)
			assert.NoError(t, err)
			expectStatus := corev1.Status{
				Components: map[string]corev1.Component{
					"fake": {
						Message: "ok",
						PodList: []corev1.PodStatus{{
							Name:              "fake",
							Namespace:         "fake",
							Status:            "Running",
							CreationTimestamp: metav1.Time{},
							RestartCount:      1,
						}},
					}},
			}
			assert.Equal(t, expectStatus, target.Status)
		},
		wantErr: false,
	}, {
		name: "HTTP check normal",
		mgr: &core.FakeManager{
			Client: fake.NewClientBuilder().WithScheme(schema).WithObjects(healthWithHTTP).Build(),
			Scheme: schema,
		},
		prepare: func() {
			gock.New("http://localhost").
				Get("/healthz").
				Reply(200).
				BodyString("ok")
		},
		req: defaultRequest,
		verify: func(t *testing.T, c client.Client) {
			target := &corev1.Health{}
			err := c.Get(context.Background(), defaultRequest.NamespacedName, target)
			assert.NoError(t, err)
			expectStatus := corev1.Status{
				Components: map[string]corev1.Component{
					"fake": {
						Message: "ok",
					}},
			}
			assert.Equal(t, expectStatus, target.Status)
		},
		wantErr: false,
	}, {
		name: "HTTP check, have error",
		mgr: &core.FakeManager{
			Client: fake.NewClientBuilder().WithScheme(schema).WithObjects(healthWithHTTP).Build(),
			Scheme: schema,
		},
		prepare: func() {
			gock.New("http://localhost").
				Get("/healthz").
				Reply(500).
				BodyString("ok")
		},
		req: defaultRequest,
		verify: func(t *testing.T, c client.Client) {
			target := &corev1.Health{}
			err := c.Get(context.Background(), defaultRequest.NamespacedName, target)
			assert.NoError(t, err)
			expectStatus := corev1.Status{
				Components: map[string]corev1.Component{
					"fake": {
						Message: "response status code '500' is not 200",
					}},
			}
			assert.Equal(t, expectStatus, target.Status)
		},
		wantErr: false,
	}, {
		name: "invalid component selector",
		mgr: &core.FakeManager{
			Client: fake.NewClientBuilder().WithScheme(schema).WithObjects(invalidSelectors).Build(),
			Scheme: schema,
		},
		req: defaultRequest,
		verify: func(t *testing.T, c client.Client) {
			target := &corev1.Health{}
			err := c.Get(context.Background(), defaultRequest.NamespacedName, target)
			assert.NoError(t, err)
			expectStatus := corev1.Status{
				Components: map[string]corev1.Component{
					"fake": {
						Message: "unable to parse requirement: <nil>: Invalid value: \"@\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
					}},
			}
			assert.Equal(t, expectStatus, target.Status)
		},
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer gock.Off()
			if tt.verify == nil {
				tt.verify = func(t *testing.T, c client.Client) {}
			}
			if tt.prepare == nil {
				tt.prepare = func() {}
			}
			tt.prepare()

			c := tt.mgr.GetClient()
			reconciler := core.NewHealthReconciler(c)
			reconciler.SetupWithManager(tt.mgr)
			assert.Equal(t, "core", reconciler.FeatureGroup())
			assert.Equal(t, "health", reconciler.FeatureName())
			assert.Equal(t, "alpha", reconciler.FeatureGate())

			_, err := reconciler.Reconcile(context.Background(), tt.req)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			tt.verify(t, c)
		})
	}
}

func TestHTTPCheck(t *testing.T) {
	tests := []struct {
		name    string
		action  *v1.HTTPGetAction
		prepare func()
		hasErr  bool
	}{{
		name:   "normal",
		action: &v1.HTTPGetAction{Path: "/healthz"},
		prepare: func() {
			gock.New("http://localhost").
				Get("/healthz").
				Reply(200).
				BodyString("ok")
		},
		hasErr: false,
	}, {
		name:   "status code is not 200",
		action: &v1.HTTPGetAction{Path: "/healthz"},
		prepare: func() {
			gock.New("http://localhost").
				Get("/healthz").
				Reply(500).
				BodyString("ok")
		},
		hasErr: true,
	}, {
		name:    "action is nil",
		action:  nil,
		prepare: func() {},
		hasErr:  false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer gock.Clean()
			tt.prepare()

			err := core.HTTPCheck(tt.action)
			if tt.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
