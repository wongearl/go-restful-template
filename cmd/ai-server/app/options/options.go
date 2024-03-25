package options

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"strings"

	aiserver "github.com/wongearl/go-restful-template/pkg/aiserver"
	aiserverconfig "github.com/wongearl/go-restful-template/pkg/aiserver/config"
	"github.com/wongearl/go-restful-template/pkg/client/cache"
	"github.com/wongearl/go-restful-template/pkg/client/informers"
	"github.com/wongearl/go-restful-template/pkg/client/k8s"
	genericoptions "github.com/wongearl/go-restful-template/pkg/server/options"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog"
)

type ServerRunOptions struct {
	ConfigFile              string
	GenericServerRunOptions *genericoptions.ServerRunOptions
	*aiserverconfig.Config
	DebugMode bool
}

func NewServerRunOptions() *ServerRunOptions {
	s := &ServerRunOptions{
		GenericServerRunOptions: genericoptions.NewServerRunOptions(),
		Config:                  aiserverconfig.New(),
	}

	return s
}

func (s *ServerRunOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("generic")
	fs.BoolVar(&s.DebugMode, "debug", false, "Don't enable this if you don't know what it means.")
	s.GenericServerRunOptions.AddFlags(fs, s.GenericServerRunOptions)
	fs = fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		fs.AddGoFlag(fl)
	})

	return fss
}

func (s *ServerRunOptions) NewAPIServer(stopCh <-chan struct{}) (*aiserver.APIServer, error) {
	apiServer := &aiserver.APIServer{
		Config: s.Config,
	}

	kubernetesClient, err := k8s.NewKubernetesClient(s.KubernetesOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client, %v", err)
	}
	apiServer.KubernetesClient = kubernetesClient

	informerFactory := informers.NewInformerFactories(kubernetesClient.Kubernetes(), kubernetesClient.ApiExtensions(), kubernetesClient.Ai())
	apiServer.InformerFactory = informerFactory

	cacheClient, err := cache.New(s.CacheOptions, stopCh)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache, error: %v", err)
	}
	apiServer.CacheClient = cacheClient

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", s.GenericServerRunOptions.InsecurePort),
	}

	if s.GenericServerRunOptions.SecurePort != 0 {
		certificate, err := tls.LoadX509KeyPair(s.GenericServerRunOptions.TlsCertFile, s.GenericServerRunOptions.TlsPrivateKey)
		if err != nil {
			return nil, err
		}
		server.TLSConfig.Certificates = []tls.Certificate{certificate}
	}

	apiServer.Server = server

	return apiServer, nil
}
