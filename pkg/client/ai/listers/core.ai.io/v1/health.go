// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/wongearl/go-restful-template/pkg/api/core.ai.io/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// HealthLister helps list Healths.
// All objects returned here must be treated as read-only.
type HealthLister interface {
	// List lists all Healths in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Health, err error)
	// Get retrieves the Health from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.Health, error)
	HealthListerExpansion
}

// healthLister implements the HealthLister interface.
type healthLister struct {
	indexer cache.Indexer
}

// NewHealthLister returns a new HealthLister.
func NewHealthLister(indexer cache.Indexer) HealthLister {
	return &healthLister{indexer: indexer}
}

// List lists all Healths in the indexer.
func (s *healthLister) List(selector labels.Selector) (ret []*v1.Health, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Health))
	})
	return ret, err
}

// Get retrieves the Health from the index for a given name.
func (s *healthLister) Get(name string) (*v1.Health, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("health"), name)
	}
	return obj.(*v1.Health), nil
}