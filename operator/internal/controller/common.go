package controller

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
)

// hashSpec returns a deterministic SHA256 hash of the given spec.
// Returns an error if marshaling fails so callers can handle it explicitly.
func hashSpec(spec any) (string, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal spec for hashing: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

// specDrifted compares two specs by hash and returns true if they differ.
// On marshal error, returns true (assume drift) so the caller reconciles.
func specDrifted(a, b any) bool {
	ha, err1 := hashSpec(a)
	hb, err2 := hashSpec(b)
	if err1 != nil || err2 != nil {
		return true
	}
	return ha != hb
}

// setOwnerReference sets the owner reference on the child object.
func setOwnerReference(owner *v1alpha1.InferenceStack, child client.Object) {
	isController := true
	blockDeletion := true
	child.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         v1alpha1.SchemeGroupVersion.String(),
			Kind:               "InferenceStack",
			Name:               owner.Name,
			UID:                owner.UID,
			Controller:         &isController,
			BlockOwnerDeletion: &blockDeletion,
		},
	})
}

// computePhase determines the aggregate phase from child statuses.
func computePhase(children []v1alpha1.ChildStatus) string {
	if len(children) == 0 {
		return v1alpha1.PhasePending
	}

	allReady := true
	anyError := false

	for _, c := range children {
		if !c.Ready {
			allReady = false
			if c.Message != "" && c.Message != "not configured" {
				anyError = true
			}
		}
	}

	if allReady {
		return v1alpha1.PhaseReady
	}
	if anyError {
		return v1alpha1.PhaseError
	}
	return v1alpha1.PhaseDegraded
}
