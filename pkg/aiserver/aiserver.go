package aiserver

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	rt "runtime"
	"time"

	"github.com/go-openapi/spec"
	"github.com/wongearl/go-restful-template/pkg/aiapis/core"
	iamapi "github.com/wongearl/go-restful-template/pkg/aiapis/iam/v1"
	"github.com/wongearl/go-restful-template/pkg/aiapis/oauth"
	"github.com/wongearl/go-restful-template/pkg/aiapis/version"
	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/authoricators/basic"
	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/authoricators/jwttoken"
	"github.com/wongearl/go-restful-template/pkg/aiserver/authentication/request/basictoken"
	"github.com/wongearl/go-restful-template/pkg/aiserver/authorization/authorizer"
	"github.com/wongearl/go-restful-template/pkg/aiserver/authorization/path"
	"github.com/wongearl/go-restful-template/pkg/aiserver/authorization/rbac"
	unionauthorizer "github.com/wongearl/go-restful-template/pkg/aiserver/authorization/union"
	apiserverconfig "github.com/wongearl/go-restful-template/pkg/aiserver/config"
	"github.com/wongearl/go-restful-template/pkg/aiserver/filters"
	"github.com/wongearl/go-restful-template/pkg/aiserver/pprof"
	"github.com/wongearl/go-restful-template/pkg/aiserver/request"
	"github.com/wongearl/go-restful-template/pkg/aiserver/swagger"
	iamiov1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	cacheclient "github.com/wongearl/go-restful-template/pkg/client/cache"
	"github.com/wongearl/go-restful-template/pkg/client/clusterclient"
	"github.com/wongearl/go-restful-template/pkg/client/informers"
	"github.com/wongearl/go-restful-template/pkg/client/k8s"
	"github.com/wongearl/go-restful-template/pkg/middles/auth"
	"github.com/wongearl/go-restful-template/pkg/middles/iam/am"
	"github.com/wongearl/go-restful-template/pkg/middles/iam/im"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/loginrecord"
	"github.com/wongearl/go-restful-template/pkg/middles/resources/v1alpha3/user"
	utilnet "github.com/wongearl/go-restful-template/pkg/utils/net"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	urlruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/request/anonymous"
	"k8s.io/apiserver/pkg/authentication/request/bearertoken"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	unionauth "k8s.io/apiserver/pkg/authentication/request/union"
)

type APIServer struct {
	Server *http.Server

	Config *apiserverconfig.Config
	// all webservice defines
	container *restful.Container
	//kubeClient is a collection of all K8S objects clientset
	KubernetesClient k8s.Client
	InformerFactory  informers.InformerFactory
	CacheClient      cacheclient.Interface
	ClusterClient    clusterclient.ClusterClients
}

func (s *APIServer) PrepareRun(stopCh <-chan struct{}) error {
	s.container = restful.NewContainer()
	s.container.Filter(logRequestAndResponse)
	s.container.Router(restful.CurlyRouter{})
	s.container.RecoverHandler(func(panicReason interface{}, httpWriter http.ResponseWriter) {
		logStackOnRecover(panicReason, httpWriter)
	})

	s.installAPIs(stopCh)

	config := restfulspec.Config{
		WebServices:                   s.container.RegisteredWebServices(),
		APIPath:                       "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject}
	s.container.Add(restfulspec.NewOpenAPIService(config))

	for _, ws := range s.container.RegisteredWebServices() {
		klog.V(2).Infof("%s", ws.RootPath())
	}

	s.Server.Handler = s.container
	s.buildHandlerChain(stopCh)
	return nil
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "ai",
			Description: "ai",
			Version:     "1.0.0",
		},
	}
}

func (s *APIServer) buildHandlerChain(stopCh <-chan struct{}) {
	requestInfoResolver := &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis", "ai-apis", "ai-api"),
		GrouplessAPIPrefixes: sets.NewString("api", "ai-api"),
		GlobalResources: []schema.GroupResource{
			iamiov1.Resource(iamiov1.ResourcePluralUser),
			iamiov1.Resource(iamiov1.ResourcePluralGlobalRole),
			iamiov1.Resource(iamiov1.ResourcePluralGlobalRoleBinding),
		},
	}

	handler := s.Server.Handler

	handler = filters.WithKubeAPIServer(handler, s.KubernetesClient.Config(), &errorResponder{})

	// this is useful for the test use cases
	if !s.Config.AuthenticationOptions.Disabled {
		var authorizers authorizer.Authorizer
		excludedPaths := []string{"/oauth/token", "/ai-apis/register.ai.io/*", "/ai-apis/config.ai.io/*", "/ai-apis/version", "/ai-apis/metrics",
			"/ai-apis/storage.ai.io/v1/s3/health",
			"/apidocs", "/apidocs/*", "/apidocs.json", "/debug/pprof"}
		pathAuthorizer, _ := path.NewAuthorizer(excludedPaths)
		amOperator := am.NewReadOnlyOperator(s.InformerFactory)
		authorizers = unionauthorizer.New(pathAuthorizer, rbac.NewRBACAuthorizer(amOperator))
		handler = filters.WithAuthorization(handler, authorizers)
	}

	loginRecorder := auth.NewLoginRecorder(s.KubernetesClient.Ai())
	// authenticators are unordered
	authn := unionauth.New(anonymous.NewAuthenticator(),
		basictoken.New(basic.NewBasicAuthenticator(auth.NewPasswordAuthenticator(s.KubernetesClient.Ai(),
			s.InformerFactory.AiSharedInformerFactory().Iam().V1().Users().Lister(),
			s.Config.AuthenticationOptions, s.Config.AiOptions), loginRecorder)),
		bearertoken.New(jwttoken.NewTokenAuthenticator(auth.NewTokenOperator(s.CacheClient, s.Config.AuthenticationOptions),
			s.InformerFactory.AiSharedInformerFactory().Iam().V1().Users().Lister())))
	handler = filters.WithAuthentication(handler, authn)

	handler = filters.WithRequestInfo(handler, requestInfoResolver)

	s.Server.Handler = handler
}

func (s *APIServer) Run(ctx context.Context) (err error) {
	err = s.waitForResourceSync(ctx)
	if err != nil {
		return err
	}

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-ctx.Done()
		_ = s.Server.Shutdown(shutdownCtx)
	}()

	klog.V(0).Infof("Start listening on %s", s.Server.Addr)
	if s.Server.TLSConfig != nil {
		err = s.Server.ListenAndServeTLS("", "")
	} else {
		err = s.Server.ListenAndServe()
	}

	return err
}

func logRequestAndResponse(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()
	chain.ProcessFilter(req, resp)

	// Always log error response
	logWithVerbose := klog.V(4)
	if resp.StatusCode() > http.StatusBadRequest {
		logWithVerbose = klog.V(0)
	}

	logWithVerbose.Infof("%s - \"%s %s %s\" %d %d %dms",
		utilnet.GetRequestIP(req.Request),
		req.Request.Method,
		req.Request.URL,
		req.Request.Proto,
		resp.StatusCode(),
		resp.ContentLength(),
		time.Since(start)/time.Millisecond,
	)
}

func logStackOnRecover(panicReason interface{}, w http.ResponseWriter) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("recover from panic situation: - %v\r\n", panicReason))
	for i := 2; ; i += 1 {
		_, file, line, ok := rt.Caller(i)
		if !ok {
			break
		}
		buffer.WriteString(fmt.Sprintf("    %s:%d\r\n", file, line))
	}
	klog.Errorln(buffer.String())

	headers := http.Header{}
	if ct := w.Header().Get("Content-Type"); len(ct) > 0 {
		headers.Set("Accept", ct)
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal server error"))
}

func (s *APIServer) waitForResourceSync(ctx context.Context) error {
	klog.V(0).Info("Start cache objects")

	stopCh := ctx.Done()

	k8sGVRs := map[schema.GroupVersion][]string{
		{Group: "", Version: "v1"}: {
			"namespaces",
			"nodes",
			"resourcequotas",
			"pods",
			"services",
			"persistentvolumeclaims",
			"secrets",
			"configmaps",
			"serviceaccounts",
		},
		{Group: "rbac.authorization.k8s.io", Version: "v1"}: {
			"roles",
			"rolebindings",
			"clusterroles",
			"clusterrolebindings",
		},
		{Group: "apps", Version: "v1"}: {
			"deployments",
			"daemonsets",
			"replicasets",
			"statefulsets",
			"controllerrevisions",
		},
		{Group: "storage.k8s.io", Version: "v1"}: {
			"storageclasses",
		},
		{Group: "batch", Version: "v1"}: {
			"jobs",
			"cronjobs",
		},
		{Group: "networking.k8s.io", Version: "v1"}: {
			"ingresses",
			"networkpolicies",
		},
		{Group: "autoscaling", Version: "v2beta2"}: {
			"horizontalpodautoscalers",
		},
	}

	if err := waitForCacheSync(s.KubernetesClient.Kubernetes().Discovery(),
		s.InformerFactory.KubernetesSharedInformerFactory(),
		func(resource schema.GroupVersionResource) (interface{}, error) {
			return s.InformerFactory.KubernetesSharedInformerFactory().ForResource(resource)
		},
		k8sGVRs, stopCh); err != nil {
		return err
	}

	alGVRs := map[schema.GroupVersion][]string{
		{Group: "iam.ai.io", Version: "v1"}: {
			"users",
			"globalroles",
			"globalrolebindings",

			"loginrecords",
		},
	}

	if err := waitForCacheSync(s.KubernetesClient.Kubernetes().Discovery(),
		s.InformerFactory.AiSharedInformerFactory(),
		func(resource schema.GroupVersionResource) (interface{}, error) {
			return s.InformerFactory.AiSharedInformerFactory().ForResource(resource)
		},
		alGVRs, stopCh); err != nil {
		return err
	}

	apiextensionsGVRs := map[schema.GroupVersion][]string{
		{Group: "apiextensions.k8s.io", Version: "v1"}: {
			"customresourcedefinitions",
		},
	}

	if err := waitForCacheSync(s.KubernetesClient.Kubernetes().Discovery(),
		s.InformerFactory.ApiExtensionSharedInformerFactory(),
		func(resource schema.GroupVersionResource) (interface{}, error) {
			return s.InformerFactory.ApiExtensionSharedInformerFactory().ForResource(resource)
		},
		apiextensionsGVRs, stopCh); err != nil {
		return err
	}
	klog.V(0).Info("Finished caching objects")
	return nil
}

func (s *APIServer) installAPIs(stop <-chan struct{}) {
	imOperator := im.NewOperator(s.KubernetesClient.Ai(),
		user.New(s.InformerFactory.AiSharedInformerFactory(),
			s.InformerFactory.KubernetesSharedInformerFactory()),
		loginrecord.New(s.InformerFactory.AiSharedInformerFactory()),
		s.Config.AuthenticationOptions)
	amOperator := am.NewOperator(s.KubernetesClient.Ai(),
		s.KubernetesClient.Kubernetes(),
		s.InformerFactory)
	rbacAuthorizer := rbac.NewRBACAuthorizer(amOperator)
	urlruntime.Must(iamapi.AddToContainer(s.container, imOperator, amOperator, s.Config.AiOptions, rbacAuthorizer))
	urlruntime.Must(oauth.AddToContainer(s.container, imOperator, s.Config.AiOptions, s.KubernetesClient.Kubernetes(),
		auth.NewTokenOperator(s.CacheClient,
			s.Config.AuthenticationOptions),
		auth.NewPasswordAuthenticator(
			s.KubernetesClient.Ai(),
			s.InformerFactory.AiSharedInformerFactory().Iam().V1().Users().Lister(),
			s.Config.AuthenticationOptions, s.Config.AiOptions),
		auth.NewLoginRecorder(s.KubernetesClient.Ai())))

	urlruntime.Must(version.AddToContainer(s.container))
	swagger.AddToContainer("docs/swagger-ui", s.container)
	pprof.AddToContainer(s.container)
	core.AddToContainer(s.container, s.KubernetesClient.Ai().CoreV1())
}

type informerForResourceFunc func(resource schema.GroupVersionResource) (interface{}, error)

func waitForCacheSync(discoveryClient discovery.DiscoveryInterface, sharedInformerFactory informers.GenericInformerFactory, informerForResourceFunc informerForResourceFunc, GVRs map[schema.GroupVersion][]string, stopCh <-chan struct{}) error {
	for groupVersion, resourceNames := range GVRs {
		var apiResourceList *v1.APIResourceList
		var err error
		err = retry.OnError(retry.DefaultRetry, func(err error) bool {
			return !errors.IsNotFound(err)
		}, func() error {
			apiResourceList, err = discoveryClient.ServerResourcesForGroupVersion(groupVersion.String())
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to fetch group version resources %s: %s", groupVersion, err)
		}
		for _, resourceName := range resourceNames {
			groupVersionResource := groupVersion.WithResource(resourceName)
			if !isResourceExists(apiResourceList.APIResources, groupVersionResource) {
				klog.Warningf("resource %s not exists in the cluster", groupVersionResource)
			} else {
				if _, err = informerForResourceFunc(groupVersionResource); err != nil {
					return fmt.Errorf("failed to create informer for %s: %s", groupVersionResource, err)
				}
			}
		}
	}
	sharedInformerFactory.Start(stopCh)
	sharedInformerFactory.WaitForCacheSync(stopCh)
	return nil
}

func isResourceExists(apiResources []v1.APIResource, resource schema.GroupVersionResource) bool {
	for _, apiResource := range apiResources {
		if apiResource.Name == resource.Resource {
			return true
		}
	}
	return false
}

type errorResponder struct{}

func (e *errorResponder) Error(w http.ResponseWriter, req *http.Request, err error) {
	klog.Error(err)
	responsewriters.InternalError(w, req, err)
}
