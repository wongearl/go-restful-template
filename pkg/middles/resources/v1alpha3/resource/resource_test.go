package resource

import (
	"testing"

	"github.com/wongearl/go-restful-template/pkg/client/informers"

	"github.com/stretchr/testify/assert"
)

func TestTryResource(t *testing.T) {
	tests := []struct {
		name         string
		clusterScope bool
		resource     string
		isNil        bool
	}{{
		name:         "deployments",
		clusterScope: false,
		resource:     "deployments",
		isNil:        false,
	}, {
		name:         "namespaces",
		clusterScope: true,
		resource:     "namespaces",
		isNil:        false,
	}, {
		name:         "fake",
		clusterScope: false,
		resource:     "fake",
		isNil:        true,
	}, {
		name:         "disks",
		clusterScope: false,
		resource:     "disks",
		isNil:        false,
	}, {
		name:         "datadisks",
		clusterScope: false,
		resource:     "datadisks",
		isNil:        false,
	}, {
		name:         "sshkeys",
		clusterScope: false,
		resource:     "sshkeys",
		isNil:        false,
	}, {
		name:         "vmcronjobs",
		clusterScope: false,
		resource:     "vmcronjobs",
		isNil:        false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := informers.NewInformerFactories(nil, nil, nil)
			resourceGetter := NewResourceGetter(factory)
			inter := resourceGetter.TryResource(tt.clusterScope, tt.resource)
			assert.Equal(t, tt.isNil, inter == nil)
			if tt.isNil {
				obj, err := resourceGetter.Get(tt.resource, "fake", "fake")
				assert.Nil(t, obj)
				assert.NotNil(t, err)

				list, err := resourceGetter.List(tt.resource, "fake", nil)
				assert.Nil(t, list)
				assert.NotNil(t, err)
			}

			assert.True(t, len(resourceGetter.clusterResourceGetters) > 0)
			assert.True(t, len(resourceGetter.namespacedResourceGetters) > 0)
		})
	}
}
