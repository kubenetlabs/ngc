package handlers

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
	mc "github.com/kubenetlabs/ngc/api/internal/multicluster"
)

func TestQueryGPUCapacity_WithGPUNodes(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gpu-node-1",
				Labels: map[string]string{"nvidia.com/gpu.product": "NVIDIA-A100-SXM4-80GB"},
			},
			Status: corev1.NodeStatus{
				Capacity: corev1.ResourceList{
					"nvidia.com/gpu": resource.MustParse("8"),
				},
				Allocatable: corev1.ResourceList{
					"nvidia.com/gpu": resource.MustParse("6"),
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gpu-node-2",
				Labels: map[string]string{"nvidia.com/gpu.product": "NVIDIA-L40S"},
			},
			Status: corev1.NodeStatus{
				Capacity: corev1.ResourceList{
					"nvidia.com/gpu": resource.MustParse("4"),
				},
				Allocatable: corev1.ResourceList{
					"nvidia.com/gpu": resource.MustParse("4"),
				},
			},
		},
	).Build()

	k8sClient := kubernetes.NewForTest(fakeClient)
	cc := &mc.ClusterClient{
		Name:      "test-cluster",
		Region:    "us-east-1",
		K8sClient: k8sClient,
	}

	cap := queryGPUCapacity(context.Background(), cc)

	if cap.TotalGPUs != 12 {
		t.Errorf("expected TotalGPUs=12, got %d", cap.TotalGPUs)
	}
	if cap.AllocatedGPUs != 2 {
		t.Errorf("expected AllocatedGPUs=2, got %d", cap.AllocatedGPUs)
	}
	if cap.GPUTypes["NVIDIA-A100-SXM4-80GB"] != 1 {
		t.Errorf("expected 1 A100 node, got %d", cap.GPUTypes["NVIDIA-A100-SXM4-80GB"])
	}
	if cap.GPUTypes["NVIDIA-L40S"] != 1 {
		t.Errorf("expected 1 L40S node, got %d", cap.GPUTypes["NVIDIA-L40S"])
	}
}

func TestQueryGPUCapacity_NoGPUNodes(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "cpu-node-1"},
			Status: corev1.NodeStatus{
				Capacity:    corev1.ResourceList{},
				Allocatable: corev1.ResourceList{},
			},
		},
	).Build()

	k8sClient := kubernetes.NewForTest(fakeClient)
	cc := &mc.ClusterClient{
		Name:      "test-cluster",
		Region:    "us-east-1",
		K8sClient: k8sClient,
	}

	cap := queryGPUCapacity(context.Background(), cc)

	if cap.TotalGPUs != 0 {
		t.Errorf("expected TotalGPUs=0, got %d", cap.TotalGPUs)
	}
	if cap.AllocatedGPUs != 0 {
		t.Errorf("expected AllocatedGPUs=0, got %d", cap.AllocatedGPUs)
	}
}

func TestQueryGPUCapacity_NilClient(t *testing.T) {
	cc := &mc.ClusterClient{
		Name:      "test-cluster",
		Region:    "us-east-1",
		K8sClient: nil,
	}

	cap := queryGPUCapacity(context.Background(), cc)

	if cap.TotalGPUs != 0 {
		t.Errorf("expected TotalGPUs=0, got %d", cap.TotalGPUs)
	}
}
