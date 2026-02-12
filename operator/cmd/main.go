package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
	controller "github.com/kubenetlabs/ngc/operator/internal/controller"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gatewayv1.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		healthProbeAddr      string
		enableLeaderElection bool
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8081", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8082", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	crlog.SetLogger(logr.FromSlogHandler(logger.Handler()))

	slog.Info("starting ngf-console operator")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: healthProbeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ngf-console-operator.f5.com",
	})
	if err != nil {
		slog.Error("unable to create manager", "error", err)
		os.Exit(1)
	}

	// Register controllers
	if err := (&controller.InferenceStackReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		slog.Error("unable to create InferenceStackReconciler", "error", err)
		os.Exit(1)
	}

	if err := (&controller.GatewayBundleReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		slog.Error("unable to create GatewayBundleReconciler", "error", err)
		os.Exit(1)
	}

	if err := (&controller.XCPublishReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		slog.Error("unable to create XCPublishReconciler", "error", err)
		os.Exit(1)
	}

	if err := (&controller.RouteWatcher{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		slog.Error("unable to create RouteWatcher", "error", err)
		os.Exit(1)
	}

	// Health probes
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		slog.Error("unable to set up health check", "error", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		slog.Error("unable to set up ready check", "error", err)
		os.Exit(1)
	}

	slog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		slog.Error("manager exited with error", "error", err)
		os.Exit(1)
	}
}
