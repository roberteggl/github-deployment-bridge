// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	appcache "github.com/roberteggl/github-deployment-bridge/internal/cache"
	"github.com/roberteggl/github-deployment-bridge/internal/config"
	"github.com/roberteggl/github-deployment-bridge/internal/deployment"
	ghclient "github.com/roberteggl/github-deployment-bridge/internal/github"
	"github.com/roberteggl/github-deployment-bridge/internal/health"
	"github.com/roberteggl/github-deployment-bridge/internal/kubernetes"
	"github.com/roberteggl/github-deployment-bridge/internal/metrics"
	"github.com/roberteggl/github-deployment-bridge/internal/registry"
	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

var (
	version  = "dev"
	commit   = "unknown"
	scheme   = runtime.NewScheme()
	setupLog = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kustomizev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		setupLog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	m := metrics.New(nil)
	probes := health.New()

	store, err := appcache.Open(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("open cache database: %w", err)
	}
	defer store.Close()

	retryCfg := retry.Config{
		MaxAttempts: cfg.RetryMaxAttempts,
		Initial:     cfg.RetryInitialBackoff,
		Max:         cfg.RetryMaxBackoff,
		Multiplier:  2,
	}

	gh, err := ghclient.NewAppClient(ghclient.Options{
		AppID:          cfg.GitHubAppID,
		InstallationID: cfg.GitHubInstallationID,
		PrivateKeyPath: cfg.GitHubPrivateKeyPath,
		BaseURL:        cfg.GitHubBaseURL,
		Metrics:        m,
		Retry:          retryCfg,
	})
	if err != nil {
		return fmt.Errorf("create github client: %w", err)
	}

	reg := registry.New(registry.Options{
		Metrics: m,
		Retry:   retryCfg,
	})

	reporter := deployment.NewReporter(cfg, store, reg, gh, m, log)

	mgrOpts := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0", // we expose metrics via chi below
		},
		HealthProbeBindAddress: cfg.ProbeAddr,
		LeaderElection:         cfg.LeaderElection,
		LeaderElectionID:       cfg.LeaderElectionID,
	}
	if cfg.WatchNamespace != "" {
		mgrOpts.Cache = ctrlcache.Options{
			DefaultNamespaces: map[string]ctrlcache.Config{
				cfg.WatchNamespace: {},
			},
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	if err := (&kubernetes.Reconciler{
		Client:   mgr.GetClient(),
		Finder:   &kubernetes.WorkloadFinder{Client: mgr.GetClient()},
		Reporter: reporter,
		Log:      log,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setup reconciler: %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("add healthz: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("add readyz: %w", err)
	}

	// Combined metrics + probe HTTP server on MetricsAddr.
	metricsSrv := newMetricsServer(cfg.MetricsAddr, probes)
	go func() {
		log.Info("starting metrics server", "addr", cfg.MetricsAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server failed", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = metricsSrv.Shutdown(shutdownCtx)
	}()

	probes.SetReady(true)
	log.Info("starting manager",
		"version", version,
		"commit", commit,
		"cluster", cfg.ClusterName,
		"environment", cfg.Environment,
		"watchNamespace", cfg.WatchNamespace,
	)

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("manager exited: %w", err)
	}
	return nil
}

func newMetricsServer(addr string, probes *health.Checker) *http.Server {
	r := chi.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	r.Mount("/", probes.Handler())
	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
