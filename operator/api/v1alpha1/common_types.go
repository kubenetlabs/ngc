package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ChildStatus tracks the state of a reconciled child resource.
type ChildStatus struct {
	// Kind is the resource kind (e.g., "InferencePool", "ConfigMap").
	Kind string `json:"kind"`
	// Name is the child resource name.
	Name string `json:"name"`
	// Ready indicates whether the child resource is in a ready state.
	Ready bool `json:"ready"`
	// Message provides additional context about the child's state.
	Message string `json:"message,omitempty"`
}

// ThresholdSpec defines a scaling threshold with metric and target value.
type ThresholdSpec struct {
	// Metric is the metric name to watch (e.g., "queue_depth", "kv_cache_pct").
	Metric string `json:"metric"`
	// Target is the threshold value that triggers scaling.
	Target int32 `json:"target"`
}

// Condition type constants for InferenceStack.
const (
	// ConditionReady indicates the overall readiness of the resource.
	ConditionReady = "Ready"
	// ConditionReconciled indicates whether the last reconciliation succeeded.
	ConditionReconciled = "Reconciled"
	// ConditionDegraded indicates the resource is partially functional.
	ConditionDegraded = "Degraded"
)

// Phase constants for InferenceStack status.
const (
	PhaseReady      = "Ready"
	PhasePending    = "Pending"
	PhaseDegraded   = "Degraded"
	PhaseError      = "Error"
	PhaseTerminating = "Terminating"
)

// Finalizer constants.
const (
	InferenceStackFinalizer        = "ngf-console.f5.com/inferencestack-finalizer"
	DistributedCloudPublishFinalizer = "ngf-console.f5.com/xc-publish-finalizer"
)

// SetCondition updates or appends a condition on the given slice.
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i, c := range *conditions {
		if c.Type == conditionType {
			if c.Status != status || c.Reason != reason || c.Message != message {
				(*conditions)[i].Status = status
				(*conditions)[i].Reason = reason
				(*conditions)[i].Message = message
				(*conditions)[i].LastTransitionTime = now
				(*conditions)[i].ObservedGeneration = 0
			}
			return
		}
	}
	*conditions = append(*conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}
