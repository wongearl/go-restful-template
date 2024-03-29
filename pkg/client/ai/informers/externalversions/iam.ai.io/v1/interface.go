// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/wongearl/go-restful-template/pkg/client/ai/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// GlobalRoles returns a GlobalRoleInformer.
	GlobalRoles() GlobalRoleInformer
	// GlobalRoleBindings returns a GlobalRoleBindingInformer.
	GlobalRoleBindings() GlobalRoleBindingInformer
	// LoginRecords returns a LoginRecordInformer.
	LoginRecords() LoginRecordInformer
	// Users returns a UserInformer.
	Users() UserInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// GlobalRoles returns a GlobalRoleInformer.
func (v *version) GlobalRoles() GlobalRoleInformer {
	return &globalRoleInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalRoleBindings returns a GlobalRoleBindingInformer.
func (v *version) GlobalRoleBindings() GlobalRoleBindingInformer {
	return &globalRoleBindingInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// LoginRecords returns a LoginRecordInformer.
func (v *version) LoginRecords() LoginRecordInformer {
	return &loginRecordInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Users returns a UserInformer.
func (v *version) Users() UserInformer {
	return &userInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
