package kubernetes

import (
	"context"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add client-go scheme: %v", err)
	}
	if err := gatewayv1.Install(scheme); err != nil {
		t.Fatalf("failed to add gateway-api scheme: %v", err)
	}
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add apiextensions scheme: %v", err)
	}
	return scheme
}

func TestDetectEdition_OSS(t *testing.T) {
	scheme := setupScheme(t)

	// Create fake client with no enterprise CRDs
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := NewForTest(fakeClient)

	edition := k8sClient.DetectEdition(context.Background())

	if edition != EditionOSS {
		t.Errorf("expected edition %s, got %s", EditionOSS, edition)
	}
}

func TestDetectEdition_Enterprise(t *testing.T) {
	scheme := setupScheme(t)

	// Create an enterprise CRD
	enterpriseCRD := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "appolicies.appprotect.f5.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "appprotect.f5.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "appolicies",
				Singular: "appolicy",
				Kind:     "APPolicy",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1beta1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
						},
					},
				},
			},
		},
	}

	// Create fake client with enterprise CRD
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(enterpriseCRD).Build()
	k8sClient := NewForTest(fakeClient)

	edition := k8sClient.DetectEdition(context.Background())

	if edition != EditionEnterprise {
		t.Errorf("expected edition %s, got %s", EditionEnterprise, edition)
	}
}

func TestDetectEdition_AlternateEnterpriseCRD(t *testing.T) {
	scheme := setupScheme(t)

	// Create a different enterprise CRD (APDoS)
	enterpriseCRD := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "apdoslogconfs.appprotectdos.f5.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "appprotectdos.f5.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "apdoslogconfs",
				Singular: "apdoslogconf",
				Kind:     "APDosLogConf",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1beta1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
						},
					},
				},
			},
		},
	}

	// Create fake client with enterprise CRD
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(enterpriseCRD).Build()
	k8sClient := NewForTest(fakeClient)

	edition := k8sClient.DetectEdition(context.Background())

	if edition != EditionEnterprise {
		t.Errorf("expected edition %s, got %s", EditionEnterprise, edition)
	}
}
