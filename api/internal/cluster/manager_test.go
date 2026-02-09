package cluster

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add client-go scheme: %v", err)
	}
	if err := gatewayv1.Install(scheme); err != nil {
		t.Fatalf("failed to add gateway-api scheme: %v", err)
	}
	return scheme
}

func testClient(t *testing.T) *kubernetes.Client {
	t.Helper()
	scheme := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	return kubernetes.NewForTest(fc)
}

func TestNewSingleCluster(t *testing.T) {
	client := testClient(t)
	mgr := NewSingleCluster(client)

	if mgr.DefaultName() != "default" {
		t.Errorf("expected default name 'default', got %q", mgr.DefaultName())
	}

	got, err := mgr.Default()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != client {
		t.Error("expected same client from Default()")
	}

	names := mgr.Names()
	if len(names) != 1 || names[0] != "default" {
		t.Errorf("expected [default], got %v", names)
	}
}

func TestManager_Get(t *testing.T) {
	clientA := testClient(t)
	clientB := testClient(t)

	mgr := NewForTest(
		map[string]*kubernetes.Client{"cluster-a": clientA, "cluster-b": clientB},
		"cluster-a",
	)

	t.Run("valid cluster", func(t *testing.T) {
		got, err := mgr.Get("cluster-a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != clientA {
			t.Error("expected clientA")
		}
	})

	t.Run("another valid cluster", func(t *testing.T) {
		got, err := mgr.Get("cluster-b")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != clientB {
			t.Error("expected clientB")
		}
	})

	t.Run("invalid cluster", func(t *testing.T) {
		_, err := mgr.Get("nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent cluster")
		}
	})
}

func TestManager_Default(t *testing.T) {
	clientA := testClient(t)
	clientB := testClient(t)

	mgr := NewForTest(
		map[string]*kubernetes.Client{"cluster-a": clientA, "cluster-b": clientB},
		"cluster-b",
	)

	got, err := mgr.Default()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != clientB {
		t.Error("expected clientB as default")
	}

	if mgr.DefaultName() != "cluster-b" {
		t.Errorf("expected default name cluster-b, got %s", mgr.DefaultName())
	}
}

func TestManager_List(t *testing.T) {
	clientA := testClient(t)
	clientB := testClient(t)

	mgr := NewForTest(
		map[string]*kubernetes.Client{"alpha": clientA, "beta": clientB},
		"alpha",
	)

	infos := mgr.List(context.Background())
	if len(infos) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(infos))
	}

	// Should be sorted by name
	if infos[0].Name != "alpha" {
		t.Errorf("expected first cluster alpha, got %s", infos[0].Name)
	}
	if infos[1].Name != "beta" {
		t.Errorf("expected second cluster beta, got %s", infos[1].Name)
	}

	// All should be connected (fake clients)
	for _, info := range infos {
		if !info.Connected {
			t.Errorf("expected cluster %s to be connected", info.Name)
		}
	}

	// Default flag
	if !infos[0].Default {
		t.Error("expected alpha to be default")
	}
	if infos[1].Default {
		t.Error("expected beta to not be default")
	}
}

func TestManager_Names(t *testing.T) {
	clientA := testClient(t)
	clientB := testClient(t)

	mgr := NewForTest(
		map[string]*kubernetes.Client{"zebra": clientA, "alpha": clientB},
		"alpha",
	)

	names := mgr.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	// Should be sorted
	if names[0] != "alpha" || names[1] != "zebra" {
		t.Errorf("expected [alpha zebra], got %v", names)
	}
}
