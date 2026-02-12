package multicluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeGroupVersion is the group version used to register ManagedCluster.
	SchemeGroupVersion = schema.GroupVersion{Group: "ngf-console.f5.com", Version: "v1alpha1"}

	// SchemeBuilder collects functions to add types to a scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds ManagedCluster types to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme

	// ManagedClusterGVR is the GroupVersionResource for ManagedCluster.
	ManagedClusterGVR = SchemeGroupVersion.WithResource("managedclusters")
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ManagedCluster{},
		&ManagedClusterList{},
	)
	return nil
}

// DeepCopyObject implements runtime.Object for ManagedCluster.
func (in *ManagedCluster) DeepCopyObject() runtime.Object {
	out := new(ManagedCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all fields into another ManagedCluster.
func (in *ManagedCluster) DeepCopyInto(out *ManagedCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)

	// Spec
	out.Spec = in.Spec
	if in.Spec.Labels != nil {
		out.Spec.Labels = make(map[string]string, len(in.Spec.Labels))
		for k, v := range in.Spec.Labels {
			out.Spec.Labels[k] = v
		}
	}
	if in.Spec.KubeconfigSecretRef != nil {
		ref := *in.Spec.KubeconfigSecretRef
		out.Spec.KubeconfigSecretRef = &ref
	}
	if in.Spec.PrometheusSecretRef != nil {
		ref := *in.Spec.PrometheusSecretRef
		out.Spec.PrometheusSecretRef = &ref
	}
	if in.Spec.AgentConfig != nil {
		ac := *in.Spec.AgentConfig
		out.Spec.AgentConfig = &ac
		if in.Spec.AgentConfig.HeartbeatIntervalSeconds != nil {
			val := *in.Spec.AgentConfig.HeartbeatIntervalSeconds
			out.Spec.AgentConfig.HeartbeatIntervalSeconds = &val
		}
	}

	// Status
	out.Status = in.Status
	if in.Status.LastHeartbeat != nil {
		t := *in.Status.LastHeartbeat
		out.Status.LastHeartbeat = &t
	}
	if in.Status.ResourceCounts != nil {
		rc := *in.Status.ResourceCounts
		out.Status.ResourceCounts = &rc
	}
	if in.Status.GPUCapacity != nil {
		gc := *in.Status.GPUCapacity
		out.Status.GPUCapacity = &gc
		if in.Status.GPUCapacity.GPUTypes != nil {
			gc.GPUTypes = make(map[string]int32, len(in.Status.GPUCapacity.GPUTypes))
			for k, v := range in.Status.GPUCapacity.GPUTypes {
				gc.GPUTypes[k] = v
			}
		}
	}
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		for i := range in.Status.Conditions {
			in.Status.Conditions[i].DeepCopyInto(&out.Status.Conditions[i])
		}
	}
}

// DeepCopyObject implements runtime.Object for ManagedClusterList.
func (in *ManagedClusterList) DeepCopyObject() runtime.Object {
	out := new(ManagedClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all fields into another ManagedClusterList.
func (in *ManagedClusterList) DeepCopyInto(out *ManagedClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]ManagedCluster, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}
