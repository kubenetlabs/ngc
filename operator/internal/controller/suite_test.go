package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
)

// testEnv holds the envtest environment for integration tests.
// These tests require the setup-envtest binary; skip gracefully if unavailable.

func setupEnvTest(t *testing.T) (client.Client, context.Context, context.CancelFunc) {
	t.Helper()

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Skipf("envtest not available: %v", err)
	}

	t.Cleanup(func() {
		if err := testEnv.Stop(); err != nil {
			t.Logf("failed to stop envtest: %v", err)
		}
	})

	if err := v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("add v1alpha1 scheme: %v", err)
	}
	if err := gatewayv1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("add gateway-api scheme: %v", err)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	if err := (&InferenceStackReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup InferenceStack controller: %v", err)
	}

	if err := (&GatewayBundleReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup GatewayBundle controller: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Logf("manager stopped: %v", err)
		}
	}()

	// Wait for cache sync
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatal("cache sync failed")
	}

	return mgr.GetClient(), ctx, cancel
}

func TestInferenceStackReconciler_CreateChildren(t *testing.T) {
	k8sClient, ctx, cancel := setupEnvTest(t)
	defer cancel()

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-create"}}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	stack := &v1alpha1.InferenceStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stack",
			Namespace: "test-create",
		},
		Spec: v1alpha1.InferenceStackSpec{
			ModelName:      "meta-llama/Llama-3-70B-Instruct",
			ServingBackend: "vllm",
			Pool: v1alpha1.InferencePoolSpec{
				GPUType:     "H100",
				GPUCount:    4,
				Replicas:    6,
				MinReplicas: 2,
				MaxReplicas: 12,
			},
			EPP: v1alpha1.EPPSpec{
				Strategy: "composite",
			},
		},
	}

	if err := k8sClient.Create(ctx, stack); err != nil {
		t.Fatalf("create InferenceStack: %v", err)
	}

	// Wait for reconciliation to create EPP ConfigMap child
	var cm corev1.ConfigMap
	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for EPP ConfigMap to be created")
		case <-ticker.C:
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-stack-epp-config",
				Namespace: "test-create",
			}, &cm)
			if err == nil {
				goto ConfigMapFound
			}
		}
	}

ConfigMapFound:
	// Verify owner reference
	if len(cm.OwnerReferences) == 0 {
		t.Fatal("EPP ConfigMap has no owner references")
	}
	if cm.OwnerReferences[0].Kind != "InferenceStack" {
		t.Errorf("expected owner kind InferenceStack, got %s", cm.OwnerReferences[0].Kind)
	}
	if cm.OwnerReferences[0].Name != "test-stack" {
		t.Errorf("expected owner name test-stack, got %s", cm.OwnerReferences[0].Name)
	}

	// Verify ConfigMap data
	if _, ok := cm.Data["epp-config.json"]; !ok {
		t.Error("EPP ConfigMap missing epp-config.json key")
	}

	// Verify status update
	var updated v1alpha1.InferenceStack
	deadline = time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for status update")
		case <-ticker.C:
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-stack",
				Namespace: "test-create",
			}, &updated); err == nil && updated.Status.ObservedSpecHash != "" {
				goto StatusUpdated
			}
		}
	}

StatusUpdated:
	if updated.Status.ObservedSpecHash == "" {
		t.Error("observedSpecHash not set")
	}
	if updated.Status.LastReconciledAt == nil {
		t.Error("lastReconciledAt not set")
	}
	if len(updated.Status.Children) == 0 {
		t.Error("no children in status")
	}

	// Find EPP ConfigMap child status
	found := false
	for _, child := range updated.Status.Children {
		if child.Kind == "ConfigMap" && child.Name == "test-stack-epp-config" {
			found = true
			if !child.Ready {
				t.Errorf("EPP ConfigMap child not ready: %s", child.Message)
			}
		}
	}
	if !found {
		t.Error("EPP ConfigMap not found in status children")
	}
}

func TestInferenceStackReconciler_SelfHeal(t *testing.T) {
	k8sClient, ctx, cancel := setupEnvTest(t)
	defer cancel()

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-heal"}}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	stack := &v1alpha1.InferenceStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "heal-stack",
			Namespace: "test-heal",
		},
		Spec: v1alpha1.InferenceStackSpec{
			ModelName:      "test-model",
			ServingBackend: "vllm",
			Pool: v1alpha1.InferencePoolSpec{
				GPUType:  "A100",
				GPUCount: 2,
				Replicas: 3,
			},
		},
	}

	if err := k8sClient.Create(ctx, stack); err != nil {
		t.Fatalf("create InferenceStack: %v", err)
	}

	// Wait for ConfigMap to exist
	var cm corev1.ConfigMap
	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for EPP ConfigMap")
		case <-ticker.C:
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "heal-stack-epp-config",
				Namespace: "test-heal",
			}, &cm); err == nil {
				goto CMCreated
			}
		}
	}

CMCreated:
	// Delete the ConfigMap (simulate drift)
	if err := k8sClient.Delete(ctx, &cm); err != nil {
		t.Fatalf("delete ConfigMap: %v", err)
	}

	// Wait for self-healing (ConfigMap should be recreated)
	deadline = time.After(90 * time.Second) // reconcile interval is 60s
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for self-heal")
		case <-ticker.C:
			var recreated corev1.ConfigMap
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "heal-stack-epp-config",
				Namespace: "test-heal",
			}, &recreated); err == nil {
				return // Self-healed
			}
		}
	}
}

func TestInferenceStackReconciler_Delete(t *testing.T) {
	k8sClient, ctx, cancel := setupEnvTest(t)
	defer cancel()

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-delete"}}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	stack := &v1alpha1.InferenceStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "del-stack",
			Namespace: "test-delete",
		},
		Spec: v1alpha1.InferenceStackSpec{
			ModelName:      "test-model",
			ServingBackend: "triton",
			Pool: v1alpha1.InferencePoolSpec{
				GPUType:  "T4",
				GPUCount: 1,
				Replicas: 1,
			},
		},
	}

	if err := k8sClient.Create(ctx, stack); err != nil {
		t.Fatalf("create InferenceStack: %v", err)
	}

	// Wait for ConfigMap
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for EPP ConfigMap")
		case <-ticker.C:
			var cm corev1.ConfigMap
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "del-stack-epp-config",
				Namespace: "test-delete",
			}, &cm); err == nil {
				goto CMExists
			}
		}
	}

CMExists:
	// Delete the InferenceStack
	if err := k8sClient.Delete(ctx, stack); err != nil {
		t.Fatalf("delete InferenceStack: %v", err)
	}

	// Verify InferenceStack is gone (finalizer should be removed)
	deadline = time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for InferenceStack deletion")
		case <-ticker.C:
			var s v1alpha1.InferenceStack
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "del-stack",
				Namespace: "test-delete",
			}, &s)
			if err != nil {
				return // Deleted
			}
		}
	}
}

// --- GatewayBundle envtest tests ---

func TestGatewayBundleReconciler_CreateGateway(t *testing.T) {
	k8sClient, ctx, cancel := setupEnvTest(t)
	defer cancel()

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-gwb-create"}}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	bundle := &v1alpha1.GatewayBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gw",
			Namespace: "test-gwb-create",
		},
		Spec: v1alpha1.GatewayBundleSpec{
			GatewayClassName: "nginx",
			Listeners: []v1alpha1.GatewayListenerSpec{
				{
					Name:     "http",
					Port:     80,
					Protocol: "HTTP",
				},
				{
					Name:     "https",
					Port:     443,
					Protocol: "HTTPS",
					Hostname: "example.com",
				},
			},
		},
	}

	if err := k8sClient.Create(ctx, bundle); err != nil {
		t.Fatalf("create GatewayBundle: %v", err)
	}

	// Wait for the Gateway child to be created
	var gw gatewayv1.Gateway
	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for Gateway child to be created")
		case <-ticker.C:
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-gw",
				Namespace: "test-gwb-create",
			}, &gw)
			if err == nil {
				goto GatewayFound
			}
		}
	}

GatewayFound:
	// Verify owner reference
	if len(gw.OwnerReferences) == 0 {
		t.Fatal("Gateway has no owner references")
	}
	if gw.OwnerReferences[0].Kind != "GatewayBundle" {
		t.Errorf("expected owner kind GatewayBundle, got %s", gw.OwnerReferences[0].Kind)
	}
	if gw.OwnerReferences[0].Name != "test-gw" {
		t.Errorf("expected owner name test-gw, got %s", gw.OwnerReferences[0].Name)
	}

	// Verify Gateway spec
	if string(gw.Spec.GatewayClassName) != "nginx" {
		t.Errorf("expected gatewayClassName nginx, got %s", gw.Spec.GatewayClassName)
	}
	if len(gw.Spec.Listeners) != 2 {
		t.Fatalf("expected 2 listeners, got %d", len(gw.Spec.Listeners))
	}
	if string(gw.Spec.Listeners[0].Name) != "http" {
		t.Errorf("expected listener name http, got %s", gw.Spec.Listeners[0].Name)
	}
	if int(gw.Spec.Listeners[0].Port) != 80 {
		t.Errorf("expected listener port 80, got %d", gw.Spec.Listeners[0].Port)
	}
	if gw.Spec.Listeners[1].Hostname == nil || string(*gw.Spec.Listeners[1].Hostname) != "example.com" {
		t.Errorf("expected listener hostname example.com")
	}

	// Verify managed-by labels
	if gw.Labels["app.kubernetes.io/managed-by"] != "ngf-console" {
		t.Error("expected managed-by label")
	}

	// Verify GatewayBundle status update
	var updated v1alpha1.GatewayBundle
	deadline = time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for GatewayBundle status update")
		case <-ticker.C:
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-gw",
				Namespace: "test-gwb-create",
			}, &updated); err == nil && updated.Status.ObservedSpecHash != "" {
				goto StatusUpdated
			}
		}
	}

StatusUpdated:
	if updated.Status.ObservedSpecHash == "" {
		t.Error("observedSpecHash not set")
	}
	if len(updated.Status.Children) == 0 {
		t.Error("no children in status")
	}

	// Verify Gateway child status
	found := false
	for _, child := range updated.Status.Children {
		if child.Kind == "Gateway" && child.Name == "test-gw" {
			found = true
			if !child.Ready {
				t.Logf("Gateway child not ready (expected in envtest): %s", child.Message)
			}
		}
	}
	if !found {
		t.Error("Gateway not found in status children")
	}
}

func TestGatewayBundleReconciler_Delete(t *testing.T) {
	k8sClient, ctx, cancel := setupEnvTest(t)
	defer cancel()

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-gwb-del"}}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	bundle := &v1alpha1.GatewayBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "del-gw",
			Namespace: "test-gwb-del",
		},
		Spec: v1alpha1.GatewayBundleSpec{
			GatewayClassName: "nginx",
			Listeners: []v1alpha1.GatewayListenerSpec{
				{Name: "http", Port: 80, Protocol: "HTTP"},
			},
		},
	}

	if err := k8sClient.Create(ctx, bundle); err != nil {
		t.Fatalf("create GatewayBundle: %v", err)
	}

	// Wait for Gateway child
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for Gateway child")
		case <-ticker.C:
			var gw gatewayv1.Gateway
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "del-gw",
				Namespace: "test-gwb-del",
			}, &gw); err == nil {
				goto GWCreated
			}
		}
	}

GWCreated:
	// Delete the GatewayBundle
	if err := k8sClient.Delete(ctx, bundle); err != nil {
		t.Fatalf("delete GatewayBundle: %v", err)
	}

	// Verify GatewayBundle is gone (finalizer removed)
	deadline = time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for GatewayBundle deletion")
		case <-ticker.C:
			var b v1alpha1.GatewayBundle
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "del-gw",
				Namespace: "test-gwb-del",
			}, &b); err != nil {
				return // Deleted
			}
		}
	}
}
