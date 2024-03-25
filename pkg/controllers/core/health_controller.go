package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "github.com/wongearl/go-restful-template/pkg/api/core.ai.io/v1"
	"github.com/wongearl/go-restful-template/pkg/controllers/common"

	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"
)

// HealthReconciler reconciles a Tool object
type HealthReconciler struct {
	client.Client
	featureGroup, featureName, featureGate string
}

// NewHealthReconciler creates an instance of common.Reconsiler
func NewHealthReconciler(client client.Client) common.FeaturedReconciler {
	return &HealthReconciler{
		Client:       client,
		featureGroup: "core",
		featureName:  "health",
		featureGate:  "alpha",
	}
}

//+kubebuilder:rbac:groups=core.ai.io,resources=healths,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.ai.io,resources=healths/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.ai.io,resources=healths/finalizers,verbs=update
//+kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch

func (r *HealthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := log.FromContext(ctx)
	health := &corev1.Health{}
	if err = r.Get(ctx, req.NamespacedName, health); err != nil {
		err = client.IgnoreNotFound(err)
		return
	}
	log.Info("Reconcile", "health", req.NamespacedName)

	defer func() {
		result = RegularReconcile(health, err)
	}()

	if health.Spec.ComponentSelectors == nil && health.Spec.ServiceProbes == nil {
		return
	}

	if health.Status.Components == nil {
		health.Status.Components = make(map[string]corev1.Component)
	}

	for key, val := range health.Spec.ComponentSelectors {
		podList := &v1.PodList{}
		var podListErr error
		var selector labels.Selector
		if selector, podListErr = labels.Parse(val); podListErr == nil {
			podListErr = r.List(ctx, podList, &client.ListOptions{LabelSelector: selector})
		}

		if podListErr != nil {
			health.Status.Components[key] = corev1.Component{
				Message: podListErr.Error(),
			}
		} else {
			podStatusList := make([]corev1.PodStatus, len(podList.Items))
			for i, pod := range podList.Items {
				podStatusList[i] = corev1.PodStatus{
					Name:              pod.Name,
					Namespace:         pod.Namespace,
					RestartCount:      getPodRestartCount(&pod),
					CreationTimestamp: pod.CreationTimestamp,
					Status:            string(pod.Status.Phase),
				}
			}
			health.Status.Components[key] = corev1.Component{
				Message: "ok",
				PodList: podStatusList,
			}
		}
	}

	for key, val := range health.Spec.ServiceProbes {
		if err = HTTPCheck(val.HTTPGet); err != nil {
			health.Status.Components[key] = corev1.Component{
				Message: err.Error(),
			}
		} else {
			health.Status.Components[key] = corev1.Component{
				Message: "ok",
			}
		}
	}
	err = r.Status().Update(ctx, health)
	return
}

func HTTPCheck(action *v1.HTTPGetAction) (err error) {
	if action == nil {
		return
	}

	if action.Scheme == "" {
		action.Scheme = v1.URISchemeHTTP
	}
	if action.Port.IntValue() == 0 {
		action.Port.Type = intstr.String
		action.Port.StrVal = "80"
	}
	if action.Host == "" {
		action.Host = "localhost"
	}
	if strings.HasPrefix(action.Path, "/") {
		action.Path = strings.TrimPrefix(action.Path, "/")
	}
	api := fmt.Sprintf("%s://%s:%d/%s", action.Scheme, action.Host, action.Port.IntValue(), action.Path)
	var resp *http.Response
	if resp, err = http.Get(api); err == nil && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("response status code '%d' is not 200", resp.StatusCode)
	}
	return
}

func getPodRestartCount(pod *v1.Pod) int {
	count := 0
	if pod.Status.ContainerStatuses != nil {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			count += int(containerStatus.RestartCount)
		}
	}
	return count
}

func RegularReconcile(health *corev1.Health, err error) (result ctrl.Result) {
	if health.Spec.Interval == 0 {
		health.Spec.Interval = time.Second * 30
	}

	if err == nil {
		result = ctrl.Result{RequeueAfter: health.Spec.Interval}
	}
	return
}

// FeatureGroup returns the feature group name
func (r *HealthReconciler) FeatureGroup() string {
	return r.featureGroup
}

// FeatureName returns the feature name
func (r *HealthReconciler) FeatureName() string {
	return r.featureName
}

// FeatureGate returns the feature gate
func (r *HealthReconciler) FeatureGate() string {
	return r.featureGate
}

// SetupWithManager sets up the controller with the Manager.
func (r *HealthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Health{}).
		Complete(r)
}
