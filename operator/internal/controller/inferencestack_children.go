package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
)

// reconcileInferencePool creates or updates the InferencePool child resource.
func (r *InferenceStackReconciler) reconcileInferencePool(ctx context.Context, stack *v1alpha1.InferenceStack) v1alpha1.ChildStatus {
	name := stack.Name + "-pool"
	log := slog.With("child", "InferencePool", "name", name)

	desired := buildDesiredInferencePool(stack, name)

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(desired.GroupVersionKind())
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: stack.Namespace}, existing)

	if errors.IsNotFound(err) {
		log.Info("creating InferencePool")
		if err := r.Create(ctx, desired); err != nil {
			log.Error("failed to create InferencePool", "error", err)
			return v1alpha1.ChildStatus{Kind: "InferencePool", Name: name, Ready: false, Message: fmt.Sprintf("create failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "InferencePool", Name: name, Ready: true, Message: "created"}
	}
	if err != nil {
		log.Error("failed to get InferencePool", "error", err)
		return v1alpha1.ChildStatus{Kind: "InferencePool", Name: name, Ready: false, Message: fmt.Sprintf("get failed: %v", err)}
	}

	// Update spec if drifted
	desiredSpec, _, _ := unstructured.NestedMap(desired.Object, "spec")
	existingSpec, _, _ := unstructured.NestedMap(existing.Object, "spec")

	if specDrifted(desiredSpec, existingSpec) {
		log.Info("InferencePool spec drifted, updating")
		existing.Object["spec"] = desired.Object["spec"]
		if err := r.Update(ctx, existing); err != nil {
			log.Error("failed to update InferencePool", "error", err)
			return v1alpha1.ChildStatus{Kind: "InferencePool", Name: name, Ready: false, Message: fmt.Sprintf("update failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "InferencePool", Name: name, Ready: true, Message: "updated"}
	}

	return v1alpha1.ChildStatus{Kind: "InferencePool", Name: name, Ready: true, Message: "in sync"}
}

// servingPort returns the default target port for a given serving backend.
func servingPort(backend string) int64 {
	switch backend {
	case "triton":
		return 8001
	case "tgi":
		return 80
	case "ollama":
		return 11434
	default: // vllm
		return 8000
	}
}

// buildDesiredInferencePool constructs the desired InferencePool unstructured object.
func buildDesiredInferencePool(stack *v1alpha1.InferenceStack, name string) *unstructured.Unstructured {
	pool := &unstructured.Unstructured{}
	pool.SetGroupVersionKind(inferencePoolGVK())
	pool.SetName(name)
	pool.SetNamespace(stack.Namespace)

	setOwnerRef(pool, stack)

	// Build selector.matchLabels
	selector := stack.Spec.Pool.Selector
	if selector == nil {
		selector = map[string]string{
			"app": stack.Name,
		}
	}
	matchLabels := make(map[string]interface{}, len(selector))
	for k, v := range selector {
		matchLabels[k] = v
	}

	// EPP service name follows the convention: <stack>-epp
	eppServiceName := stack.Name + "-epp"

	spec := map[string]interface{}{
		"targetPorts": []interface{}{
			map[string]interface{}{
				"number": servingPort(stack.Spec.ServingBackend),
			},
		},
		"selector": map[string]interface{}{
			"matchLabels": matchLabels,
		},
		"endpointPickerRef": map[string]interface{}{
			"group":       "",
			"kind":        "Service",
			"name":        eppServiceName,
			"failureMode": "FailClose",
			"port": map[string]interface{}{
				"number": int64(9002),
			},
		},
	}

	pool.Object["spec"] = spec

	pool.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "ngf-console",
		"ngf-console.f5.com/stack":     stack.Name,
	})

	return pool
}

// reconcileEPPConfig creates or updates the EPP ConfigMap child resource.
func (r *InferenceStackReconciler) reconcileEPPConfig(ctx context.Context, stack *v1alpha1.InferenceStack) v1alpha1.ChildStatus {
	name := stack.Name + "-epp-config"
	log := slog.With("child", "ConfigMap", "name", name)

	desired := buildDesiredEPPConfigMap(stack, name)

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: stack.Namespace}, existing)

	if errors.IsNotFound(err) {
		log.Info("creating EPP ConfigMap")
		if err := r.Create(ctx, desired); err != nil {
			log.Error("failed to create EPP ConfigMap", "error", err)
			return v1alpha1.ChildStatus{Kind: "ConfigMap", Name: name, Ready: false, Message: fmt.Sprintf("create failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "ConfigMap", Name: name, Ready: true, Message: "created"}
	}
	if err != nil {
		log.Error("failed to get EPP ConfigMap", "error", err)
		return v1alpha1.ChildStatus{Kind: "ConfigMap", Name: name, Ready: false, Message: fmt.Sprintf("get failed: %v", err)}
	}

	// Update data if drifted
	if specDrifted(existing.Data, desired.Data) {
		log.Info("EPP ConfigMap drifted, updating")
		existing.Data = desired.Data
		if err := r.Update(ctx, existing); err != nil {
			log.Error("failed to update EPP ConfigMap", "error", err)
			return v1alpha1.ChildStatus{Kind: "ConfigMap", Name: name, Ready: false, Message: fmt.Sprintf("update failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "ConfigMap", Name: name, Ready: true, Message: "updated"}
	}

	return v1alpha1.ChildStatus{Kind: "ConfigMap", Name: name, Ready: true, Message: "in sync"}
}

// buildDesiredEPPConfigMap constructs the desired EPP ConfigMap.
func buildDesiredEPPConfigMap(stack *v1alpha1.InferenceStack, name string) *corev1.ConfigMap {
	strategy := stack.Spec.EPP.Strategy
	if strategy == "" {
		strategy = "least_queue"
	}

	eppConfig := map[string]interface{}{
		"strategy":  strategy,
		"poolName":  stack.Name + "-pool",
		"modelName": stack.Spec.ModelName,
	}

	if stack.Spec.EPP.Weights != nil {
		eppConfig["weights"] = map[string]interface{}{
			"queueDepth":     stack.Spec.EPP.Weights.QueueDepth,
			"kvCache":        stack.Spec.EPP.Weights.KVCache,
			"prefixAffinity": stack.Spec.EPP.Weights.PrefixAffinity,
		}
	}

	configJSON, _ := json.MarshalIndent(eppConfig, "", "  ")

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stack.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "ngf-console",
				"ngf-console.f5.com/stack":     stack.Name,
			},
		},
		Data: map[string]string{
			"epp-config.json": string(configJSON),
		},
	}

	// Set owner reference
	isController := true
	blockDeletion := true
	cm.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         v1alpha1.SchemeGroupVersion.String(),
			Kind:               "InferenceStack",
			Name:               stack.Name,
			UID:                stack.UID,
			Controller:         &isController,
			BlockOwnerDeletion: &blockDeletion,
		},
	})

	return cm
}

// reconcileAutoscaler creates or updates the KEDA ScaledObject child resource.
func (r *InferenceStackReconciler) reconcileAutoscaler(ctx context.Context, stack *v1alpha1.InferenceStack) v1alpha1.ChildStatus {
	name := stack.Name + "-scaler"
	if stack.Spec.Autoscaling == nil {
		return v1alpha1.ChildStatus{Kind: "ScaledObject", Name: name, Ready: true, Message: "not configured"}
	}

	log := slog.With("child", "ScaledObject", "name", name)

	desired := buildDesiredScaledObject(stack, name)

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(kedaScaledObjectGVK())
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: stack.Namespace}, existing)

	if errors.IsNotFound(err) {
		log.Info("creating ScaledObject")
		if err := r.Create(ctx, desired); err != nil {
			log.Error("failed to create ScaledObject", "error", err)
			return v1alpha1.ChildStatus{Kind: "ScaledObject", Name: name, Ready: false, Message: fmt.Sprintf("create failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "ScaledObject", Name: name, Ready: true, Message: "created"}
	}
	if err != nil {
		log.Error("failed to get ScaledObject", "error", err)
		return v1alpha1.ChildStatus{Kind: "ScaledObject", Name: name, Ready: false, Message: fmt.Sprintf("get failed: %v", err)}
	}

	desiredSpec, _, _ := unstructured.NestedMap(desired.Object, "spec")
	existingSpec, _, _ := unstructured.NestedMap(existing.Object, "spec")

	if specDrifted(desiredSpec, existingSpec) {
		log.Info("ScaledObject spec drifted, updating")
		existing.Object["spec"] = desired.Object["spec"]
		if err := r.Update(ctx, existing); err != nil {
			log.Error("failed to update ScaledObject", "error", err)
			return v1alpha1.ChildStatus{Kind: "ScaledObject", Name: name, Ready: false, Message: fmt.Sprintf("update failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "ScaledObject", Name: name, Ready: true, Message: "updated"}
	}

	return v1alpha1.ChildStatus{Kind: "ScaledObject", Name: name, Ready: true, Message: "in sync"}
}

// buildDesiredScaledObject constructs the KEDA ScaledObject unstructured object.
func buildDesiredScaledObject(stack *v1alpha1.InferenceStack, name string) *unstructured.Unstructured {
	so := &unstructured.Unstructured{}
	so.SetGroupVersionKind(kedaScaledObjectGVK())
	so.SetName(name)
	so.SetNamespace(stack.Namespace)
	so.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "ngf-console",
		"ngf-console.f5.com/stack":     stack.Name,
	})

	setOwnerRef(so, stack)

	cooldown := int64(300)
	if stack.Spec.Autoscaling.CooldownSeconds > 0 {
		cooldown = int64(stack.Spec.Autoscaling.CooldownSeconds)
	}

	triggers := make([]interface{}, 0, len(stack.Spec.Autoscaling.Thresholds))
	for _, t := range stack.Spec.Autoscaling.Thresholds {
		triggers = append(triggers, map[string]interface{}{
			"type": "prometheus",
			"metadata": map[string]interface{}{
				"serverAddress": "http://prometheus.monitoring:9090",
				"metricName":    t.Metric,
				"threshold":     strconv.Itoa(int(t.Target)),
				"query":         fmt.Sprintf(`avg(%s{pool="%s"})`, t.Metric, stack.Name+"-pool"),
			},
		})
	}

	so.Object["spec"] = map[string]interface{}{
		"scaleTargetRef": map[string]interface{}{
			"name": stack.Name + "-pool",
		},
		"minReplicaCount":  int64(stack.Spec.Pool.MinReplicas),
		"maxReplicaCount":  int64(stack.Spec.Pool.MaxReplicas),
		"cooldownPeriod":   cooldown,
		"pollingInterval":  int64(15),
		"triggers":         triggers,
	}

	return so
}

// reconcileHTTPRoute creates or updates the HTTPRoute child resource.
func (r *InferenceStackReconciler) reconcileHTTPRoute(ctx context.Context, stack *v1alpha1.InferenceStack) v1alpha1.ChildStatus {
	name := stack.Name + "-route"
	if stack.Spec.HTTPRoute == nil {
		return v1alpha1.ChildStatus{Kind: "HTTPRoute", Name: name, Ready: true, Message: "not configured"}
	}

	log := slog.With("child", "HTTPRoute", "name", name)

	desired := buildDesiredHTTPRoute(stack, name)

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(httpRouteGVK())
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: stack.Namespace}, existing)

	if errors.IsNotFound(err) {
		log.Info("creating HTTPRoute")
		if err := r.Create(ctx, desired); err != nil {
			log.Error("failed to create HTTPRoute", "error", err)
			return v1alpha1.ChildStatus{Kind: "HTTPRoute", Name: name, Ready: false, Message: fmt.Sprintf("create failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "HTTPRoute", Name: name, Ready: true, Message: "created"}
	}
	if err != nil {
		log.Error("failed to get HTTPRoute", "error", err)
		return v1alpha1.ChildStatus{Kind: "HTTPRoute", Name: name, Ready: false, Message: fmt.Sprintf("get failed: %v", err)}
	}

	desiredSpec, _, _ := unstructured.NestedMap(desired.Object, "spec")
	existingSpec, _, _ := unstructured.NestedMap(existing.Object, "spec")

	if specDrifted(desiredSpec, existingSpec) {
		log.Info("HTTPRoute spec drifted, updating")
		existing.Object["spec"] = desired.Object["spec"]
		if err := r.Update(ctx, existing); err != nil {
			log.Error("failed to update HTTPRoute", "error", err)
			return v1alpha1.ChildStatus{Kind: "HTTPRoute", Name: name, Ready: false, Message: fmt.Sprintf("update failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "HTTPRoute", Name: name, Ready: true, Message: "updated"}
	}

	return v1alpha1.ChildStatus{Kind: "HTTPRoute", Name: name, Ready: true, Message: "in sync"}
}

// buildDesiredHTTPRoute constructs the HTTPRoute unstructured object.
func buildDesiredHTTPRoute(stack *v1alpha1.InferenceStack, name string) *unstructured.Unstructured {
	hr := &unstructured.Unstructured{}
	hr.SetGroupVersionKind(httpRouteGVK())
	hr.SetName(name)
	hr.SetNamespace(stack.Namespace)
	hr.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "ngf-console",
		"ngf-console.f5.com/stack":     stack.Name,
	})

	setOwnerRef(hr, stack)

	hostnames := make([]interface{}, 0, len(stack.Spec.HTTPRoute.Hostnames))
	for _, h := range stack.Spec.HTTPRoute.Hostnames {
		hostnames = append(hostnames, h)
	}

	gwNamespace := stack.Namespace
	if stack.Spec.HTTPRoute.GatewayNamespace != "" {
		gwNamespace = stack.Spec.HTTPRoute.GatewayNamespace
	}

	parentRef := map[string]interface{}{
		"group": "gateway.networking.k8s.io",
		"kind":  "Gateway",
		"name":  stack.Spec.HTTPRoute.GatewayRef,
	}
	if gwNamespace != stack.Namespace {
		parentRef["namespace"] = gwNamespace
	}

	spec := map[string]interface{}{
		"parentRefs": []interface{}{parentRef},
		"rules": []interface{}{
			map[string]interface{}{
				"backendRefs": []interface{}{
					map[string]interface{}{
						"group": "inference.networking.x-k8s.io",
						"kind":  "InferencePool",
						"name":  stack.Name + "-pool",
					},
				},
			},
		},
	}
	if len(hostnames) > 0 {
		spec["hostnames"] = hostnames
	}

	hr.Object["spec"] = spec

	return hr
}

// reconcileDCGMExporter creates or updates the DCGM DaemonSet child resource.
func (r *InferenceStackReconciler) reconcileDCGMExporter(ctx context.Context, stack *v1alpha1.InferenceStack) v1alpha1.ChildStatus {
	name := stack.Name + "-dcgm"
	if stack.Spec.DCGM == nil || !stack.Spec.DCGM.Enabled {
		return v1alpha1.ChildStatus{Kind: "DaemonSet", Name: name, Ready: true, Message: "not configured"}
	}

	log := slog.With("child", "DaemonSet", "name", name)

	desired := buildDesiredDCGMDaemonSet(stack, name)

	existing := &appsv1.DaemonSet{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: stack.Namespace}, existing)

	if errors.IsNotFound(err) {
		log.Info("creating DCGM DaemonSet")
		if err := r.Create(ctx, desired); err != nil {
			log.Error("failed to create DCGM DaemonSet", "error", err)
			return v1alpha1.ChildStatus{Kind: "DaemonSet", Name: name, Ready: false, Message: fmt.Sprintf("create failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "DaemonSet", Name: name, Ready: true, Message: "created"}
	}
	if err != nil {
		log.Error("failed to get DCGM DaemonSet", "error", err)
		return v1alpha1.ChildStatus{Kind: "DaemonSet", Name: name, Ready: false, Message: fmt.Sprintf("get failed: %v", err)}
	}

	// Check if image drifted
	if len(existing.Spec.Template.Spec.Containers) > 0 &&
		existing.Spec.Template.Spec.Containers[0].Image != desired.Spec.Template.Spec.Containers[0].Image {
		log.Info("DCGM DaemonSet image drifted, updating")
		existing.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
		if err := r.Update(ctx, existing); err != nil {
			log.Error("failed to update DCGM DaemonSet", "error", err)
			return v1alpha1.ChildStatus{Kind: "DaemonSet", Name: name, Ready: false, Message: fmt.Sprintf("update failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "DaemonSet", Name: name, Ready: true, Message: "updated"}
	}

	ready := existing.Status.NumberReady > 0
	msg := "in sync"
	if !ready {
		msg = fmt.Sprintf("waiting for pods (%d ready)", existing.Status.NumberReady)
	}
	return v1alpha1.ChildStatus{Kind: "DaemonSet", Name: name, Ready: ready, Message: msg}
}

// buildDesiredDCGMDaemonSet constructs the DCGM exporter DaemonSet.
func buildDesiredDCGMDaemonSet(stack *v1alpha1.InferenceStack, name string) *appsv1.DaemonSet {
	image := "nvcr.io/nvidia/k8s/dcgm-exporter:3.3.5-3.4.1-ubuntu22.04"
	if stack.Spec.DCGM.Image != "" {
		image = stack.Spec.DCGM.Image
	}

	labels := map[string]string{
		"app.kubernetes.io/managed-by": "ngf-console",
		"ngf-console.f5.com/stack":     stack.Name,
		"app":                          name,
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stack.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "dcgm-exporter",
							Image: image,
							Ports: []corev1.ContainerPort{
								{Name: "metrics", ContainerPort: 9400, Protocol: corev1.ProtocolTCP},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot: boolPtr(true),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_ADMIN"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set owner reference
	isController := true
	blockDeletion := true
	ds.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         v1alpha1.SchemeGroupVersion.String(),
			Kind:               "InferenceStack",
			Name:               stack.Name,
			UID:                stack.UID,
			Controller:         &isController,
			BlockOwnerDeletion: &blockDeletion,
		},
	})

	return ds
}

// setOwnerRef sets the standard owner reference on an unstructured object.
func setOwnerRef(obj *unstructured.Unstructured, stack *v1alpha1.InferenceStack) {
	isController := true
	blockDeletion := true
	obj.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         v1alpha1.SchemeGroupVersion.String(),
			Kind:               "InferenceStack",
			Name:               stack.Name,
			UID:                stack.UID,
			Controller:         &isController,
			BlockOwnerDeletion: &blockDeletion,
		},
	})
}

func boolPtr(b bool) *bool { return &b }

// GVK helpers

func inferencePoolGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "inference.networking.k8s.io",
		Version: "v1",
		Kind:    "InferencePool",
	}
}

func kedaScaledObjectGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "keda.sh",
		Version: "v1alpha1",
		Kind:    "ScaledObject",
	}
}

func httpRouteGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	}
}
