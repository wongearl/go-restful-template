package api

import (
	"encoding/json"
	"fmt"
	"github.com/wongearl/go-restful-template/pkg/constants"
	"github.com/wongearl/go-restful-template/pkg/utils/net"
	"github.com/wongearl/go-restful-template/pkg/utils/sliceutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/emicklei/go-restful"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Result represents a HTTP response to front-end
type Result[T interface{}] struct {
	// Message is the hunman readable message for the front-end.
	// This message is 'OK' when code is '0', or is the error message.
	Message string `json:"message"`
	// Code represents the result, 0 means success, others mean failure
	Code int `json:"code"`
	// Data is the response from HTTP server.
	// Data could be slice or a single object.
	Data interface{} `json:"data,omitempty"`
	// Page carries the pagination of a list result
	Page *ResultPage `json:"page,omitempty"`
	err  error       `json:"-"`
}

// ResultPage is the result pagination object
type ResultPage struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// NewResult returns a list result
func NewResult[T interface{}]() *Result[T] {
	return &Result[T]{
		Message: "OK",
		Code:    0,
	}
}

// NewStringResult creates a string result
func NewStringResult() *Result[string] {
	return &Result[string]{
		Message: "OK",
		Code:    0,
	}
}

// NewEmptyResult creates a non-data result
func NewEmptyResult() *Result[string] {
	return &Result[string]{
		Message: "OK",
		Code:    0,
	}
}

// WithList puts the object list into result
func (r *Result[T]) WithList(list []T) *Result[T] {
	return r.WithListAndFilter(list, nil)
}

func (r *Result[T]) WithObjectAsJSONString(obj interface{}) *Result[T] {
	data, err := json.Marshal(obj)
	if err == nil {
		r.Data = string(data)
	} else {
		r.Data = err.Error()
	}
	return r
}

type metadataFilter func(metav1.ObjectMetaAccessor) bool

// Querier represents an interface for querying parameter
type Querier interface {
	QueryParameter(string) string
}

// WithListAndFilter set and filter the list
func (r *Result[T]) WithListAndFilter(list []T, querier Querier) *Result[T] {
	var metadataFilters []metadataFilter
	if querier != nil {
		if alias := querier.QueryParameter("aliasname"); alias != "" {
			metadataFilters = append(metadataFilters, func(oma metav1.ObjectMetaAccessor) bool {
				var annos map[string]string
				if annos = oma.GetObjectMeta().GetAnnotations(); annos != nil {
					val := annos[constants.NameAnnotationkey]
					return strings.Contains(val, alias)
				}
				return false
			})
		}
		if name := querier.QueryParameter("name"); name != "" {
			metadataFilters = append(metadataFilters, func(oma metav1.ObjectMetaAccessor) bool {
				return strings.Contains(oma.GetObjectMeta().GetName(), name)
			})
		}
		if names := querier.QueryParameter("names"); names != "" {
			metadataFilters = append(metadataFilters, func(oma metav1.ObjectMetaAccessor) bool {
				items := strings.Split(names, ",")
				for _, name := range items {
					if strings.Contains(oma.GetObjectMeta().GetName(), name) {
						return true
					}
				}
				return false
			})
		}
	}

	var toDels []int
	for i, item := range list {
		if reflect.TypeOf(item).Kind() == reflect.Pointer {
			if !r.match(list[i], metadataFilters) {
				toDels = append(toDels, i)
				continue
			}
			r.cleanManagedFields(list[i])
		} else {
			if !r.matchFromInterface(&list[i], metadataFilters) {
				toDels = append(toDels, i)
				continue
			}
			r.cleanManagedFieldsFromInterface(&list[i])
		}
	}

	resultList := sliceutil.RemoveSubSlice(list, toDels)
	r.Data = resultList
	if len(resultList) > 0 {
		r.Page = &ResultPage{
			Total: len(resultList),
		}
	}
	return r
}

func (r *Result[T]) matchFromInterface(obj *T, metadataFilters []metadataFilter) (match bool) {
	if len(metadataFilters) == 0 || metadataFilters == nil {
		match = true
		return
	}

	switch meta := any(obj).(type) {
	case metav1.ObjectMetaAccessor:
		for _, filter := range metadataFilters {
			if match = filter(meta); !match {
				return
			}
		}
	}
	return
}

func (r *Result[T]) match(obj T, metadataFilters []metadataFilter) (match bool) {
	if len(metadataFilters) == 0 {
		match = true
		return
	}

	switch meta := any(obj).(type) {
	case metav1.ObjectMetaAccessor:
		for _, filter := range metadataFilters {
			if match = filter(meta); !match {
				return
			}
		}
	}
	return
}

// WithObject puts a single object into result
func (r *Result[T]) WithObject(obj T) *Result[T] {
	r.cleanManagedFields(obj)
	r.Data = obj
	return r
}

// WithError sets the error message
func (r *Result[T]) WithError(err error) *Result[T] {
	r.err = err
	if err != nil {
		r.Message = err.Error()
		if r.Code == 0 {
			r.Code = -1
		}
		r.Data = nil
	}
	return r
}

// WriteTo writes data to target Writer
func (r *Result[T]) WriteTo(resp *restful.Response) {
	statusCode := http.StatusOK
	if r.Code != 0 {
		statusCode = http.StatusBadRequest
	}
	if r.err != nil {
		fmt.Println("error happend", r.err)
	}
	if err := resp.WriteHeaderAndJson(statusCode, r, net.ContentTypeJSON); err != nil {
		fmt.Println("failed to write response", err)
		_ = resp.WriteHeaderAndJson(http.StatusBadRequest, CommonSingleResult[error]{Data: err}, net.ContentTypeJSON)
	}
}

// CommonSingleResultis the sub-result of the common one.
// This struct aims to get the single object.
type CommonSingleResult[T interface{}] struct {
	Result[T] `json:"inline"`
	Data      T `json:"data"`
}

// ParseResult parses the result with a single object
func ParseResult[T interface{}](data []byte) (result *CommonSingleResult[T], err error) {
	result = &CommonSingleResult[T]{}
	err = json.Unmarshal(data, result)
	return
}

// GetData returns the data
func (r *CommonSingleResult[T]) GetData() T {
	return r.Data
}

// CommonListResult is the sub-result of the common one.
// This struct aims to get list items.
type CommonListResult[T interface{}] struct {
	Result[T] `json:"inline"`
	Data      []T `json:"data"`
}

// ParseListResult parses the data to result object
func ParseListResult[T interface{}](data []byte) (result *CommonListResult[T], err error) {
	result = &CommonListResult[T]{}
	err = json.Unmarshal(data, result)
	return
}

// GetList returns the all items.
func (r *CommonListResult[T]) GetList() (items []T) {
	items = r.Data
	return
}

func (r *Result[T]) cleanManagedFields(obj T) {
	switch meta := any(obj).(type) {
	case metav1.ObjectMetaAccessor:
		vt := reflect.TypeOf(meta)
		if vt.Kind() == reflect.Ptr && reflect.ValueOf(meta).IsNil() {
			return
		}
		meta.GetObjectMeta().SetManagedFields(nil)
	}
}

func (r *Result[T]) cleanManagedFieldsFromInterface(obj *T) {
	switch meta := any(obj).(type) {
	case metav1.ObjectMetaAccessor:
		meta.GetObjectMeta().SetManagedFields(nil)
	}
}

type ListResult struct {
	Items      []interface{} `json:"items"`
	TotalItems int           `json:"totalItems"`
}

type ResourceQuota struct {
	Namespace string                     `json:"namespace" description:"namespace"`
	Data      corev1.ResourceQuotaStatus `json:"data" description:"resource quota status"`
}

type NamespacedResourceQuota struct {
	Namespace string `json:"namespace,omitempty"`

	Data struct {
		corev1.ResourceQuotaStatus

		// quota left status, do the math on the side, cause it's
		// a lot easier with go-client library
		Left corev1.ResourceList `json:"left,omitempty"`
	} `json:"data,omitempty"`
}

type Router struct {
	RouterType  string            `json:"type"`
	Annotations map[string]string `json:"annotations"`
}

type GitCredential struct {
	RemoteUrl string                  `json:"remoteUrl" description:"git server url"`
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty" description:"auth secret reference"`
}

type RegistryCredential struct {
	Username   string `json:"username" description:"username"`
	Password   string `json:"password" description:"password"`
	ServerHost string `json:"serverhost" description:"registry server host"`
}

type Workloads struct {
	Namespace string                 `json:"namespace" description:"the name of the namespace"`
	Count     map[string]int         `json:"data" description:"the number of unhealthy workloads"`
	Items     map[string]interface{} `json:"items,omitempty" description:"unhealthy workloads"`
}

type ClientType string

const (
	ClientKubernetes  ClientType = "Kubernetes"
	ClientApplication ClientType = "Application"

	StatusOK = "ok"
)

// List of all resource kinds supported by the UI.
const (
	WorkspaceNone = ""
	ClusterNone   = ""
)
