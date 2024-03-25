package constants

const (
	APIVersion          = "v1alpha1"
	KubeSystemNamespace = "kube-system"
	MonitoringNamespace = "kube-monitoring"
	ApplicationName     = "app.kubernetes.io/name"
	ApplicationVersion  = "app.kubernetes.io/version"
	AlertingTag         = "Alerting"

	ClusterMetricsTag   = "Cluster Metrics"
	NodeMetricsTag      = "Node Metrics"
	NamespaceMetricsTag = "Namespace Metrics"
	PodMetricsTag       = "Pod Metrics"
	VMIMetricsTag       = "VMI Metrics"
	PVCMetricsTag       = "PVC Metrics"
	IngressMetricsTag   = "Ingress Metrics"
	ContainerMetricsTag = "Container Metrics"
	WorkloadMetricsTag  = "Workload Metrics"
	WorkspaceMetricsTag = "Workspace Metrics"
	ComponentMetricsTag = "Component Metrics"
	CustomMetricsTag    = "Custom Metrics"

	DisplayNameAnnotationKey = "ai.io/alias-name"
	NamespaceLabelKey        = "ai.io/namespace"
	CreatorAnnotationKey     = "ai.io/creator"
	UsernameAnnotationKey    = "ai.io/username"
	PasswordAnnotationKey    = "ai.io/password"
	RegisterAnnotationKey    = "ai.io/register"
	RegisterSecretKey        = "ai@123456789"
	WorkspaceLabelKey        = "ai.io/workspace"

	DescriptionAnnotationkey = "ai.io/description"
	NameAnnotationkey        = "ai.io/name"
)
