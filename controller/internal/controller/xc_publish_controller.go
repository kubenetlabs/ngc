package controller

import (
	"context"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// DistributedCloudPublish is a placeholder type for the ngf-console.f5.com/v1alpha1
// DistributedCloudPublish custom resource. This will be replaced by a generated type.
type DistributedCloudPublish struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DistributedCloudPublishSpec   `json:"spec,omitempty"`
	Status DistributedCloudPublishStatus `json:"status,omitempty"`
}

// DistributedCloudPublishSpec defines the desired state of DistributedCloudPublish.
type DistributedCloudPublishSpec struct {
	HTTPRouteRef     string                 `json:"httpRouteRef,omitempty"`
	InferencePoolRef string                 `json:"inferencePoolRef,omitempty"`
	DistributedCloud DistributedCloudConfig `json:"distributedCloud,omitempty"`
}

// DistributedCloudConfig holds the Distributed Cloud configuration.
type DistributedCloudConfig struct {
	Tenant         string         `json:"tenant,omitempty"`
	Namespace      string         `json:"namespace,omitempty"`
	WAFPolicy      string         `json:"wafPolicy,omitempty"`
	BotDefense     BotDefense     `json:"botDefense,omitempty"`
	DDoSProtection DDoSProtection `json:"ddosProtection,omitempty"`
	PublicHostname string         `json:"publicHostname,omitempty"`
	TLS            TLSConfig      `json:"tls,omitempty"`
	OriginPool     OriginPool     `json:"originPool,omitempty"`
	RateLimiting   RateLimiting   `json:"rateLimiting,omitempty"`
	MultiRegion    MultiRegion    `json:"multiRegion,omitempty"`
}

// BotDefense configures bot defense settings.
type BotDefense struct {
	Enabled bool `json:"enabled,omitempty"`
}

// DDoSProtection configures DDoS protection settings.
type DDoSProtection struct {
	Enabled bool `json:"enabled,omitempty"`
}

// TLSConfig configures TLS settings.
type TLSConfig struct {
	SecretRef string `json:"secretRef,omitempty"`
}

// OriginPool configures origin pool settings.
type OriginPool struct {
	Name string `json:"name,omitempty"`
}

// RateLimiting configures rate limiting settings.
type RateLimiting struct {
	Enabled         bool   `json:"enabled,omitempty"`
	RequestsPerUnit int32  `json:"requestsPerUnit,omitempty"`
	Unit            string `json:"unit,omitempty"`
}

// MultiRegion configures multi-region settings.
type MultiRegion struct {
	Enabled bool     `json:"enabled,omitempty"`
	Regions []string `json:"regions,omitempty"`
}

// DistributedCloudPublishStatus defines the observed state of DistributedCloudPublish.
type DistributedCloudPublishStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// DistributedCloudPublishList contains a list of DistributedCloudPublish.
type DistributedCloudPublishList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DistributedCloudPublish `json:"items"`
}

// DeepCopyObject implements runtime.Object for DistributedCloudPublish.
func (in *DistributedCloudPublish) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(DistributedCloudPublish)
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.TypeMeta = in.TypeMeta
	out.Spec = in.Spec
	out.Status = in.Status
	// Deep copy slices in nested structs
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
	if in.Spec.DistributedCloud.MultiRegion.Regions != nil {
		out.Spec.DistributedCloud.MultiRegion.Regions = make([]string, len(in.Spec.DistributedCloud.MultiRegion.Regions))
		copy(out.Spec.DistributedCloud.MultiRegion.Regions, in.Spec.DistributedCloud.MultiRegion.Regions)
	}
	return out
}

// DeepCopyObject implements runtime.Object for DistributedCloudPublishList.
func (in *DistributedCloudPublishList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(DistributedCloudPublishList)
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]DistributedCloudPublish, len(in.Items))
		for i := range in.Items {
			out.Items[i] = *in.Items[i].DeepCopyObject().(*DistributedCloudPublish)
		}
	}
	return out
}

var (
	// SchemeGroupVersion is the group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: "ngf-console.f5.com", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionResource scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&DistributedCloudPublish{}, &DistributedCloudPublishList{})
}

// XCPublishReconciler reconciles DistributedCloudPublish objects.
type XCPublishReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles reconciliation of DistributedCloudPublish resources.
func (r *XCPublishReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	slog.Info("reconciling DistributedCloudPublish", "name", req.Name, "namespace", req.Namespace)

	var publish DistributedCloudPublish
	if err := r.Get(ctx, req.NamespacedName, &publish); err != nil {
		slog.Error("unable to fetch DistributedCloudPublish", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	slog.Info("found DistributedCloudPublish",
		"name", publish.Name,
		"httpRouteRef", publish.Spec.HTTPRouteRef,
		"tenant", publish.Spec.DistributedCloud.Tenant,
	)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *XCPublishReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&DistributedCloudPublish{}).
		Complete(r)
}
