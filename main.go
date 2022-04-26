/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	aqua "github.com/aquasecurity/starboard/pkg/apis/aquasecurity/v1alpha1"

	"github.com/giantswarm/starboard-exporter/controllers"
	"github.com/giantswarm/starboard-exporter/controllers/configauditreport"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	err := aqua.AddToScheme(scheme)
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("error registering scheme: %s", err))
	}

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var maxJitterPercent int
	var probeAddr string
	targetLabels := []controllers.VulnerabilityLabel{}
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.IntVar(&maxJitterPercent, "max-jitter-percent", 10,
		"Spreads out re-queue interval of reports by +/- this amount to spread load.")

	// Read and parse target-labels into known VulnerabilityLabels.
	flag.Func("target-labels",
		"A comma-separated list of labels to be exposed per-vulnerability. Alias 'all' is supported.",
		func(input string) error {
			items := strings.Split(input, ",")
			for _, i := range items {
				if i == controllers.LabelGroupAll {
					// Special case for "all".
					targetLabels = appendIfNotExists(targetLabels, controllers.LabelsForGroup(controllers.LabelGroupAll))
					continue
				}

				label, ok := controllers.LabelWithName(i)
				if !ok {
					err := errors.New("invalidConfigError")
					setupLog.Error(err, fmt.Sprintf("unknown target label %s", i))
					return err
				}
				targetLabels = appendIfNotExists(targetLabels, []controllers.VulnerabilityLabel{label})
			}

			setupLog.Info(fmt.Sprintf("Using %d target labels: %v", len(targetLabels), targetLabels))
			return nil
		})

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "58aff8fc.giantswarm",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.VulnerabilityReportReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("VulnerabilityReport"),
		MaxJitterPercent: maxJitterPercent,
		Scheme:           mgr.GetScheme(),
		TargetLabels:     targetLabels,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VulnerabilityReport")
		os.Exit(1)
	}

	if err = (&configauditreport.ConfigAuditReportReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("ConfigAuditReport"),
		MaxJitterPercent: maxJitterPercent,
		Scheme:           mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConfigAuditReport")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func appendIfNotExists(base []controllers.VulnerabilityLabel, items []controllers.VulnerabilityLabel) []controllers.VulnerabilityLabel {
	result := base
	contained := make(map[string]bool)

	for _, existingLabelName := range controllers.LabelNamesForList(base) {
		contained[existingLabelName] = true
	}

	for _, newItem := range items {
		if !contained[newItem.Name] {
			result = append(result, newItem)
			contained[newItem.Name] = true
		}
	}

	return result
}
