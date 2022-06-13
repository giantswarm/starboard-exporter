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
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash/v2"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	aqua "github.com/aquasecurity/starboard/pkg/apis/aquasecurity/v1alpha1"

	"github.com/giantswarm/starboard-exporter/controllers"
	"github.com/giantswarm/starboard-exporter/controllers/ciskubebenchreport"
	"github.com/giantswarm/starboard-exporter/controllers/configauditreport"
	"github.com/giantswarm/starboard-exporter/controllers/vulnerabilityreport"
	"github.com/giantswarm/starboard-exporter/utils"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

type hasher struct{}

func (h hasher) Sum64(data []byte) uint64 {
	// TODO: Investigate hash function options.
	return xxhash.Sum64(data)
}

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
	var podIPString string
	var probeAddr string
	var serviceName string
	var serviceNamespace string
	targetLabels := []vulnerabilityreport.VulnerabilityLabel{}
	reportLabels := []ciskubebenchreport.ReportLabel{}

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.IntVar(&maxJitterPercent, "max-jitter-percent", 10,
		"Spreads out re-queue interval of reports by +/- this amount to spread load.")

	flag.StringVar(&podIPString, "pod-ip", "", "The IP address of the current Pod/instance used when sharding reports.")

	flag.StringVar(&serviceName, "service-name", controllers.DefaultServiceName, "When sharding reports, the service endpoints for this service will be used to find peers.")
	flag.StringVar(&serviceNamespace, "service-namespace", "", "When sharding reports, the service endpoints in this namespace will be used to find peers.")

	// Read and parse target-labels into known VulnerabilityLabels.
	flag.Func("target-labels",
		"A comma-separated list of labels to be exposed per-vulnerability. Alias 'all' is supported.",
		func(input string) error {
			items := strings.Split(input, ",")
			for _, i := range items {
				if i == vulnerabilityreport.LabelGroupAll {
					// Special case for "all".
					targetLabels = appendIfNotExists(targetLabels, vulnerabilityreport.LabelsForGroup(vulnerabilityreport.LabelGroupAll))
					continue
				}

				label, ok := vulnerabilityreport.LabelWithName(i)
				if !ok {
					err := errors.New("invalidConfigError")
					return err
				}
				targetLabels = appendIfNotExists(targetLabels, []vulnerabilityreport.VulnerabilityLabel{label})
			}

			// If exposing detail metrics, we must always include the report name in order to delete them by name later.
			reportNameLabel, _ := vulnerabilityreport.LabelWithName("report_name")
			targetLabels = appendIfNotExists(targetLabels, []vulnerabilityreport.VulnerabilityLabel{reportNameLabel})

			return nil
		})

	flag.Func("report-labels",
		"A comma-separated list of labels to be exposed per-report. Alias 'all' is supported.",
		func(input string) error {
			items := strings.Split(input, ",")
			for _, i := range items {
				if i == ciskubebenchreport.LabelGroupAll {
					// Special case for "all".
					reportLabels = appendIfNotExistsCIS(reportLabels, ciskubebenchreport.LabelsForGroup(ciskubebenchreport.LabelGroupAll))
					continue
				}

				label, ok := ciskubebenchreport.LabelWithName(i)
				if !ok {
					err := errors.New("invalidConfigError")
					return err
				}
				reportLabels = appendIfNotExistsCIS(reportLabels, []ciskubebenchreport.ReportLabel{label})
			}

			return nil
		})

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	podIP := net.ParseIP(podIPString)
	if podIP == nil {
		setupLog.Error(nil, fmt.Sprintf("invalid pod IP %s", podIPString))
		os.Exit(1)
	}

	if serviceNamespace == "" {
		setupLog.Error(nil, "service namespace must not be empty")
		os.Exit(1)
	}

	setupLog.Info(fmt.Sprintf("this is exporter instance %s", podIP.String()))

	// Print target labels.
	if len(targetLabels) > 0 {
		tl := []string{}
		for _, l := range targetLabels {
			tl = append(tl, l.Name)
		}
		setupLog.Info(fmt.Sprintf("Using %d target labels: %v", len(tl), tl))
	}

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

	// Set up consistent hashing to shard reports over all of our exporters.
	// This is arbitrarily based on the assumption that 97 exporters is a reasonable maximum for now.
	// This could be made configurable in the future if actual usage requires it.
	consistentCfg := consistent.Config{
		PartitionCount:    97,
		ReplicationFactor: 20,
		Load:              1.25,
		Hasher:            hasher{},
	}

	peerRing := utils.BuildPeerHashRing(consistentCfg, podIP.String(), serviceName, serviceNamespace)

	// Create and start the informer which will keep the endpoints in sync in our ring.
	stopInformer := make(chan struct{})
	defer close(stopInformer)

	informerLog := ctrl.Log.WithName("informer").WithName("Endpoints")
	inf := utils.BuildPeerInformer(stopInformer, peerRing, consistentCfg, informerLog)
	go inf.Run(stopInformer)

	// Wait for the ring to be synced for the first time so we can use it immediately.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		if peerRing.MemberCount() > 0 || ctx.Err() != nil {
			break
		}
	}

	setupLog.Info(fmt.Sprintf("found %d exporters in %s service", peerRing.MemberCount(), peerRing.ServiceName))

	if err = (&vulnerabilityreport.VulnerabilityReportReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("VulnerabilityReport"),
		MaxJitterPercent: maxJitterPercent,
		Scheme:           mgr.GetScheme(),
		ShardHelper:      peerRing,
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
		ShardHelper:      peerRing,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConfigAuditReport")
		os.Exit(1)
	}

	if err = (&ciskubebenchreport.CISKubeBenchReportReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("CISKubeBenchReport"),
		MaxJitterPercent: maxJitterPercent,
		Scheme:           mgr.GetScheme(),
		TargetLabels:     reportLabels,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CISKubeBenchReport")
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

	shutdownLog := ctrl.Log.WithName("shutdownHook")
	defer shutdownRequeue(mgr.GetClient(), shutdownLog, podIP.String())

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func shutdownRequeue(c client.Client, log logr.Logger, podIP string) {
	log.Info(fmt.Sprintf("attempting to re-queue reports for instance %s", podIP))

	vulnerabilityreport.RequeueReportsForPod(c, log, podIP)

	configauditreport.RequeueReportsForPod(c, log, podIP)

	ciskubebenchreport.RequeueReportsForPod(c, log, podIP)
}

func appendIfNotExists(base []vulnerabilityreport.VulnerabilityLabel, items []vulnerabilityreport.VulnerabilityLabel) []vulnerabilityreport.VulnerabilityLabel {
	result := base
	contained := make(map[string]bool)

	for _, existingLabelName := range vulnerabilityreport.LabelNamesForList(base) {
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

func appendIfNotExistsCIS(base []ciskubebenchreport.ReportLabel, items []ciskubebenchreport.ReportLabel) []ciskubebenchreport.ReportLabel {
	result := base
	contained := make(map[string]bool)

	for _, existingLabelName := range ciskubebenchreport.LabelNamesForList(base) {
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
