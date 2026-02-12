package controller

import (
	"testing"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
)

func TestHashSpec_Deterministic(t *testing.T) {
	spec := v1alpha1.InferenceStackSpec{
		ModelName:      "meta-llama/Llama-3-70B-Instruct",
		ServingBackend: "vllm",
		Pool: v1alpha1.InferencePoolSpec{
			GPUType:  "H100",
			GPUCount: 4,
			Replicas: 6,
		},
	}

	h1, err1 := hashSpec(spec)
	h2, err2 := hashSpec(spec)

	if err1 != nil {
		t.Fatalf("hashSpec returned error: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("hashSpec returned error: %v", err2)
	}
	if h1 == "" {
		t.Fatal("hashSpec returned empty string")
	}
	if h1 != h2 {
		t.Errorf("hashSpec not deterministic: %s != %s", h1, h2)
	}
}

func TestHashSpec_ChangesOnDiff(t *testing.T) {
	spec1 := v1alpha1.InferenceStackSpec{
		ModelName:      "meta-llama/Llama-3-70B-Instruct",
		ServingBackend: "vllm",
		Pool: v1alpha1.InferencePoolSpec{
			GPUType:  "H100",
			GPUCount: 4,
			Replicas: 6,
		},
	}

	spec2 := spec1
	spec2.Pool.Replicas = 8

	h1, err1 := hashSpec(spec1)
	h2, err2 := hashSpec(spec2)

	if err1 != nil {
		t.Fatalf("hashSpec returned error: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("hashSpec returned error: %v", err2)
	}
	if h1 == h2 {
		t.Error("hashSpec should differ for different specs")
	}
}

func TestComputePhase_AllReady(t *testing.T) {
	children := []v1alpha1.ChildStatus{
		{Kind: "InferencePool", Name: "test-pool", Ready: true},
		{Kind: "ConfigMap", Name: "test-epp-config", Ready: true},
	}

	phase := computePhase(children)
	if phase != v1alpha1.PhaseReady {
		t.Errorf("expected Ready, got %s", phase)
	}
}

func TestComputePhase_Empty(t *testing.T) {
	phase := computePhase(nil)
	if phase != v1alpha1.PhasePending {
		t.Errorf("expected Pending for empty children, got %s", phase)
	}
}

func TestComputePhase_SomeNotReady(t *testing.T) {
	children := []v1alpha1.ChildStatus{
		{Kind: "InferencePool", Name: "test-pool", Ready: true},
		{Kind: "ConfigMap", Name: "test-epp-config", Ready: false, Message: "not configured"},
	}

	// "not configured" is treated as non-error degradation
	phase := computePhase(children)
	if phase != v1alpha1.PhaseDegraded {
		t.Errorf("expected Degraded, got %s", phase)
	}
}

func TestComputePhase_Error(t *testing.T) {
	children := []v1alpha1.ChildStatus{
		{Kind: "InferencePool", Name: "test-pool", Ready: true},
		{Kind: "ConfigMap", Name: "test-epp-config", Ready: false, Message: "create failed: permission denied"},
	}

	phase := computePhase(children)
	if phase != v1alpha1.PhaseError {
		t.Errorf("expected Error, got %s", phase)
	}
}

func TestComputePhase_NotConfiguredIsReady(t *testing.T) {
	children := []v1alpha1.ChildStatus{
		{Kind: "InferencePool", Name: "test-pool", Ready: true},
		{Kind: "ConfigMap", Name: "test-epp-config", Ready: true},
		{Kind: "ScaledObject", Name: "test-scaler", Ready: true, Message: "not configured"},
		{Kind: "HTTPRoute", Name: "test-route", Ready: true, Message: "not configured"},
		{Kind: "DaemonSet", Name: "test-dcgm", Ready: true, Message: "not configured"},
	}

	phase := computePhase(children)
	if phase != v1alpha1.PhaseReady {
		t.Errorf("expected Ready (stubs marked ready), got %s", phase)
	}
}
