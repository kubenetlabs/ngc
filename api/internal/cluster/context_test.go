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

func TestClientContext_RoundTrip(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := gatewayv1.Install(scheme); err != nil {
		t.Fatal(err)
	}

	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	client := kubernetes.NewForTest(fc)

	ctx := WithClient(context.Background(), client)
	got := ClientFromContext(ctx)
	if got != client {
		t.Error("expected same client from context")
	}
}

func TestClientContext_Nil(t *testing.T) {
	got := ClientFromContext(context.Background())
	if got != nil {
		t.Error("expected nil client from empty context")
	}
}

func TestClusterNameContext_RoundTrip(t *testing.T) {
	ctx := WithClusterName(context.Background(), "production")
	got := ClusterNameFromContext(ctx)
	if got != "production" {
		t.Errorf("expected production, got %q", got)
	}
}

func TestClusterNameContext_Empty(t *testing.T) {
	got := ClusterNameFromContext(context.Background())
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
