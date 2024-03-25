package v1alpha3

import (
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/wongearl/go-restful-template/pkg/aiserver/query"
	"github.com/wongearl/go-restful-template/pkg/api"
)

type Interface interface {
	// Get retrieves a single object by its namespace and name
	Get(namespace, name string) (runtime.Object, error)

	// List retrieves a collection of objects matches given query
	List(namespace string, query *query.Query) (*api.ListResult, error)
}

// ExtraLabelSelectable allows add more label selectors before listing
type ExtraLabelSelectable interface {
	Interface
	WithRequirements(requirements []labels.Requirement)
}

// CompareFunc return true is left great than right
type CompareFunc func(runtime.Object, runtime.Object, query.Field) bool

type FilterFunc func(runtime.Object, query.Filter) bool

type TransformFunc func(runtime.Object) runtime.Object

type CustomResource interface {
	metav1.Object
	runtime.Object
}

func DefaultObjectList[T CustomResource](objects []T, q *query.Query, compareFunc CompareFunc,
	filterFunc FilterFunc, transformFuncs ...TransformFunc) *api.ListResult {
	var result []runtime.Object
	for _, object := range objects {
		object.SetManagedFields(nil)
		if object.GetAnnotations() != nil {
			delete(object.GetAnnotations(), "kubectl.kubernetes.io/last-applied-configuration")
		}
		result = append(result, object)
	}
	return DefaultList(result, q, compareFunc, filterFunc, transformFuncs...)
}

func DefaultList(objects []runtime.Object, q *query.Query, compareFunc CompareFunc, filterFunc FilterFunc, transformFuncs ...TransformFunc) *api.ListResult {
	// selected matched ones
	var filtered []runtime.Object
	for _, object := range objects {
		selected := true
		for field, value := range q.Filters {
			if !filterFunc(object, query.Filter{Field: field, Value: value}) {
				selected = false
				break
			}
		}

		if selected {
			for _, transform := range transformFuncs {
				object = transform(object)
			}
			filtered = append(filtered, object)
		}
	}

	// sort by sortBy field
	sort.Slice(filtered, func(i, j int) bool {
		if !q.Ascending {
			return compareFunc(filtered[i], filtered[j], q.SortBy)
		}
		return !compareFunc(filtered[i], filtered[j], q.SortBy)
	})

	total := len(filtered)

	if q.Pagination == nil {
		q.Pagination = query.NoPagination
	}

	start, end := q.Pagination.GetValidPagination(total)

	return &api.ListResult{
		TotalItems: len(filtered),
		Items:      objectsToInterfaces(filtered[start:end]),
	}
}

// DefaultObjectMetaCompare return true is left great than right
func DefaultObjectMetaCompare(left, right metav1.ObjectMeta, sortBy query.Field) bool {
	switch sortBy {
	// ?sortBy=name
	case query.FieldName:
		return strings.Compare(left.Name, right.Name) > 0
	//	?sortBy=creationTimestamp
	default:
		fallthrough
	case query.FieldCreateTime:
		fallthrough
	case query.FieldCreationTimeStamp:
		// compare by name if creation timestamp is equal
		if left.CreationTimestamp.Equal(&right.CreationTimestamp) {
			return strings.Compare(left.Name, right.Name) > 0
		}
		return left.CreationTimestamp.After(right.CreationTimestamp.Time)
	}
}

// Default metadata filter
func DefaultObjectMetaFilter(item metav1.ObjectMeta, filter query.Filter) bool {
	switch filter.Field {
	case query.FieldNames:
		for _, name := range strings.Split(string(filter.Value), ",") {
			if item.Name == name {
				return true
			}
		}
		return false
	// /namespaces?page=1&limit=10&name=default
	case query.FieldName:
		return strings.Contains(item.Name, string(filter.Value))
		// /namespaces?page=1&limit=10&uid=a8a8d6cf-f6a5-4fea-9c1b-e57610115706
	case query.FieldUID:
		return strings.Compare(string(item.UID), string(filter.Value)) == 0
		// /deployments?page=1&limit=10&namespace=ai-system
	case query.FieldNamespace:
		return strings.Compare(item.Namespace, string(filter.Value)) == 0
		// /namespaces?page=1&limit=10&ownerReference=a8a8d6cf-f6a5-4fea-9c1b-e57610115706
	case query.FieldOwnerReference:
		for _, ownerReference := range item.OwnerReferences {
			if strings.Compare(string(ownerReference.UID), string(filter.Value)) == 0 {
				return true
			}
		}
		return false
		// /namespaces?page=1&limit=10&ownerKind=Workspace
	case query.FieldOwnerKind:
		for _, ownerReference := range item.OwnerReferences {
			if strings.Compare(ownerReference.Kind, string(filter.Value)) == 0 {
				return true
			}
		}
		return false
		// /namespaces?page=1&limit=10&annotation=runtime
	case query.FieldAnnotation:
		return labelMatch(item.Annotations, string(filter.Value))
		// /namespaces?page=1&limit=10&label=ai.io/workspace:system-workspace
	case query.FieldLabel:
		return labelMatch(item.Labels, string(filter.Value))
	default:
		return false
	}
}

func labelMatch(m map[string]string, filter string) bool {
	labelSelector, err := labels.Parse(filter)
	if err != nil {
		klog.Warningf("invalid labelSelector %s: %s", filter, err)
		return false
	}
	return labelSelector.Matches(labels.Set(m))
}

func objectsToInterfaces(objs []runtime.Object) []interface{} {
	res := make([]interface{}, 0)
	for _, obj := range objs {
		res = append(res, obj)
	}
	return res
}
