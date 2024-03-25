package api

import (
	"errors"
	"github.com/wongearl/go-restful-template/pkg/constants"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResultWriteTo(t *testing.T) {
	tests := []struct {
		name   string
		call   func(resp *restful.Response)
		expect string
	}{{
		name: "string type for the list",
		call: func(resp *restful.Response) {
			strTypeResultList := NewResult[string]()
			strTypeResultList.WithList([]string{"one", "two"}).WriteTo(resp)
		},
		expect: expectStrResult,
	}, {
		name: "int type for the list",
		call: func(resp *restful.Response) {
			intTypeResultList := NewResult[int]()
			intTypeResultList.WithList([]int{1, 2, 3}).WriteTo(resp)
		},
		expect: expectintResult,
	}, {
		name: "with error",
		call: func(resp *restful.Response) {
			intTypeResultList := NewResult[int]()
			intTypeResultList.WithError(errors.New("fake")).WriteTo(resp)
		},
		expect: `{
 "message": "fake",
 "code": -1
}`,
	}, {
		name: "with a single object, clean managedFields from objectMeta",
		call: func(resp *restful.Response) {
			result := NewResult[*metav1.ObjectMeta]()
			result.WithObject(&metav1.ObjectMeta{
				Name:      "fake",
				Namespace: "default",
				ManagedFields: []metav1.ManagedFieldsEntry{{
					Manager: "fake-manager",
				}},
			}).WriteTo(resp)
		},
		expect: expectObjectMeta,
	}, {
		name: "clean managedFields of a list",
		call: func(resp *restful.Response) {
			result := NewResult[*metav1.ObjectMeta]()
			result.WithList([]*metav1.ObjectMeta{{
				Name:      "bar",
				Namespace: "default",
				ManagedFields: []metav1.ManagedFieldsEntry{{
					Manager: "fake-manager",
				}},
			}, {
				Name:      "foo",
				Namespace: "default",
				ManagedFields: []metav1.ManagedFieldsEntry{{
					Manager: "fake-manager",
				}},
			}}).WriteTo(resp)
		},
		expect: expectObjectMetaList,
	}, {
		name: "clean managedFields of a list, the items are struct instead of point",
		call: func(resp *restful.Response) {
			result := NewResult[metav1.ObjectMeta]()
			result.WithList([]metav1.ObjectMeta{{
				Name:      "bar",
				Namespace: "default",
				ManagedFields: []metav1.ManagedFieldsEntry{{
					Manager: "fake-manager",
				}},
			}, {
				Name:      "foo",
				Namespace: "default",
				ManagedFields: []metav1.ManagedFieldsEntry{{
					Manager: "fake-manager",
				}},
			}}).WriteTo(resp)
		},
		expect: expectObjectMetaList,
	}, {
		name: "empty result",
		call: func(resp *restful.Response) {
			NewEmptyResult().WriteTo(resp)
		},
		expect: `{
 "message": "OK",
 "code": 0
}`,
	}, {
		name: "error result",
		call: func(resp *restful.Response) {
			NewEmptyResult().WithError(errors.New("error")).WriteTo(resp)
		},
		expect: `{
 "message": "error",
 "code": -1
}`,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpWriter := httptest.NewRecorder()
			resp := restful.NewResponse(httpWriter)
			tt.call(resp)
			assert.Equal(t, tt.expect, httpWriter.Body.String(), httpWriter.Body.String())
		})
	}
}

func TestResultParse(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{{
		name: "list result",
		test: func(t *testing.T) {
			result, err := ParseListResult[metav1.ObjectMeta]([]byte(expectObjectMetaList))
			if assert.Nil(t, err) {
				assert.Equal(t, 2, len(result.GetList()))
				assert.Equal(t, "bar", result.GetList()[0].Name)
				assert.Equal(t, "foo", result.GetList()[1].Name)
			}
		},
	}, {
		name: "single object result",
		test: func(t *testing.T) {
			result, err := ParseResult[metav1.ObjectMeta]([]byte(expectObjectMeta))
			if assert.Nil(t, err) {
				assert.Equal(t, "fake", result.GetData().Name)
			}
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}
}

func TestWithListAndFilter(t *testing.T) {
	tests := []struct {
		name    string
		result  *Result[metav1.ObjectMeta]
		list    []metav1.ObjectMeta
		querier Querier
		expect  string
	}{{
		name:   "alias is not match, but name is match",
		result: NewResult[metav1.ObjectMeta](),
		list: []metav1.ObjectMeta{{
			Name:        "foo",
			Annotations: nil,
		}},
		querier: NewFakeQuerier(map[string]string{
			"aliasname": "fake",
			"name":      "foo",
		}),
		expect: `{
 "message": "OK",
 "code": 0,
 "data": []
}`,
	}, {
		name:   "alias is match, but name is not match",
		result: NewResult[metav1.ObjectMeta](),
		list: []metav1.ObjectMeta{{
			Name: "foo",
			Annotations: map[string]string{
				constants.NameAnnotationkey: "foo-alias",
			},
		}},
		querier: NewFakeQuerier(map[string]string{
			"aliasname": "fake-alias",
			"name":      "fake",
		}),
		expect: `{
 "message": "OK",
 "code": 0,
 "data": []
}`,
	}, {
		name:   "alias and name are match, struct type",
		result: NewResult[metav1.ObjectMeta](),
		list: []metav1.ObjectMeta{{
			Name: "foo",
			Annotations: map[string]string{
				constants.NameAnnotationkey: "foo-alias",
			},
		}},
		querier: NewFakeQuerier(map[string]string{
			"aliasname": "foo-alias",
			"name":      "foo",
		}),
		expect: `{
 "message": "OK",
 "code": 0,
 "data": [
  {
   "name": "foo",
   "creationTimestamp": null,
   "annotations": {
    "ai.io/name": "foo-alias"
   }
  }
 ],
 "page": {
  "total": 1,
  "page": 0,
  "limit": 0
 }
}`,
	}, {
		name:   "filter with multiple names",
		result: NewResult[metav1.ObjectMeta](),
		list: []metav1.ObjectMeta{{
			Name: "foo1",
		}, {
			Name: "foo2",
		}, {
			Name: "foo3",
		}},
		querier: NewFakeQuerier(map[string]string{
			"names": "foo1,foo2",
		}),
		expect: `{
 "message": "OK",
 "code": 0,
 "data": [
  {
   "name": "foo1",
   "creationTimestamp": null
  },
  {
   "name": "foo2",
   "creationTimestamp": null
  }
 ],
 "page": {
  "total": 2,
  "page": 0,
  "limit": 0
 }
}`,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpWriter := httptest.NewRecorder()
			resp := restful.NewResponse(httpWriter)
			tt.result.WithListAndFilter(tt.list, tt.querier).WriteTo(resp)
			assert.Equal(t, tt.expect, httpWriter.Body.String(), httpWriter.Body.String())
		})
	}

	testsInter := []struct {
		name    string
		result  *Result[*metav1.ObjectMeta]
		list    []*metav1.ObjectMeta
		querier Querier
		expect  string
	}{{
		name:   "alias is not match, but name is match",
		result: NewResult[*metav1.ObjectMeta](),
		list: []*metav1.ObjectMeta{{
			Name:        "foo",
			Annotations: nil,
		}},
		querier: NewFakeQuerier(map[string]string{
			"aliasname": "fake",
			"name":      "foo",
		}),
		expect: `{
 "message": "OK",
 "code": 0,
 "data": []
}`,
	}, {
		name:   "alias and name are match, interface type",
		result: NewResult[*metav1.ObjectMeta](),
		list: []*metav1.ObjectMeta{{
			Name: "foo",
			Annotations: map[string]string{
				constants.NameAnnotationkey: "foo-alias",
			},
		}},
		querier: NewFakeQuerier(map[string]string{
			"aliasname": "foo-alias",
			"name":      "foo",
		}),
		expect: `{
 "message": "OK",
 "code": 0,
 "data": [
  {
   "name": "foo",
   "creationTimestamp": null,
   "annotations": {
    "ai.io/name": "foo-alias"
   }
  }
 ],
 "page": {
  "total": 1,
  "page": 0,
  "limit": 0
 }
}`,
	}}
	for _, tt := range testsInter {
		t.Run(tt.name, func(t *testing.T) {
			httpWriter := httptest.NewRecorder()
			resp := restful.NewResponse(httpWriter)
			tt.result.WithListAndFilter(tt.list, tt.querier).WriteTo(resp)
			assert.Equal(t, tt.expect, httpWriter.Body.String(), httpWriter.Body.String())
		})
	}
}

var expectObjectMeta = `{
 "message": "OK",
 "code": 0,
 "data": {
  "name": "fake",
  "namespace": "default",
  "creationTimestamp": null
 }
}`

var expectObjectMetaList = `{
 "message": "OK",
 "code": 0,
 "data": [
  {
   "name": "bar",
   "namespace": "default",
   "creationTimestamp": null
  },
  {
   "name": "foo",
   "namespace": "default",
   "creationTimestamp": null
  }
 ],
 "page": {
  "total": 2,
  "page": 0,
  "limit": 0
 }
}`

var expectStrResult = `{
 "message": "OK",
 "code": 0,
 "data": [
  "one",
  "two"
 ],
 "page": {
  "total": 2,
  "page": 0,
  "limit": 0
 }
}`

var expectintResult = `{
 "message": "OK",
 "code": 0,
 "data": [
  1,
  2,
  3
 ],
 "page": {
  "total": 3,
  "page": 0,
  "limit": 0
 }
}`
