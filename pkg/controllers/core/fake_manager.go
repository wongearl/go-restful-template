package core

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// FakeManager is for the test purpose
type FakeManager struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// Add is a fake method
func (f *FakeManager) Add(manager.Runnable) error {
	return nil
}

// Elected is a fake method
func (f *FakeManager) Elected() <-chan struct{} {
	return nil
}

// SetFields is a fake method
func (f *FakeManager) SetFields(interface{}) error {
	return nil
}

// AddMetricsExtraHandler is a fake method
func (f *FakeManager) AddMetricsExtraHandler(path string, handler http.Handler) error {
	return nil
}

// AddHealthzCheck is a fake method
func (f *FakeManager) AddHealthzCheck(string, healthz.Checker) error {
	return nil
}

// AddReadyzCheck is a fake method
func (f *FakeManager) AddReadyzCheck(string, healthz.Checker) error {
	return nil
}

// Start is a fake method
func (f *FakeManager) Start(ctx context.Context) error {
	return nil
}

// GetConfig is a fake method
func (f *FakeManager) GetConfig() *rest.Config {
	return nil
}

// GetScheme is a fake method
func (f *FakeManager) GetScheme() *runtime.Scheme {
	return f.Scheme
}

// GetClient is a fake method
func (f *FakeManager) GetClient() client.Client {
	return f.Client
}

// GetFieldIndexer is a fake method
func (f *FakeManager) GetFieldIndexer() client.FieldIndexer {
	return nil
}

// GetCache is a fake method
func (f *FakeManager) GetCache() cache.Cache {
	return nil
}

// GetEventRecorderFor is a fake method
func (f *FakeManager) GetEventRecorderFor(name string) record.EventRecorder {
	return nil
}

// GetRESTMapper is a fake method
func (f *FakeManager) GetRESTMapper() meta.RESTMapper {
	return meta.FirstHitRESTMapper{}
}

// GetAPIReader is a fake method
func (f *FakeManager) GetAPIReader() client.Reader {
	return f.Client
}

// GetWebhookServer is a fake method
func (f *FakeManager) GetWebhookServer() *webhook.Server {
	return nil
}

// GetLogger is a fake method
func (f *FakeManager) GetLogger() logr.Logger {
	return logr.New(log.NullLogSink{})
}

// GetControllerOptions is a fake method
func (f *FakeManager) GetControllerOptions() (spec v1alpha1.ControllerConfigurationSpec) {
	return
}
