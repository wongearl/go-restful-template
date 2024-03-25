package core

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v1 "github.com/wongearl/go-restful-template/pkg/api/core.ai.io/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestFakeManager(t *testing.T) {
	schema, err := v1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	client := clientfake.NewClientBuilder().WithScheme(schema).Build()

	fake := FakeManager{
		Client: client,
		Scheme: schema,
	}
	assert.Nil(t, fake.Add(nil))
	assert.Nil(t, fake.Elected())
	assert.Nil(t, fake.SetFields(nil))
	assert.Nil(t, fake.AddMetricsExtraHandler("", nil))
	assert.Nil(t, fake.AddHealthzCheck("", nil))
	assert.Nil(t, fake.AddReadyzCheck("", nil))
	assert.Nil(t, fake.Start(context.Background()))
	assert.Nil(t, fake.GetConfig())
	assert.Equal(t, schema, fake.GetScheme())
	assert.Equal(t, client, fake.GetClient())
	assert.Nil(t, fake.GetFieldIndexer())
	assert.Nil(t, fake.GetCache())
	assert.Nil(t, fake.GetEventRecorderFor(""))
	assert.Equal(t, meta.FirstHitRESTMapper{}, fake.GetRESTMapper())
	assert.Equal(t, client, fake.GetAPIReader())
	assert.Nil(t, fake.GetWebhookServer())
	assert.Equal(t, logr.New(log.NullLogSink{}), fake.GetLogger())
	assert.NotNil(t, fake.GetControllerOptions())
}
