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
	"net"
	"os"
	"reflect"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	aqua "github.com/aquasecurity/starboard/pkg/apis/aquasecurity/v1alpha1"

	"github.com/giantswarm/starboard-exporter/controllers/configauditreport"
	"github.com/giantswarm/starboard-exporter/controllers/vulnerabilityreport"
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
	var podIPString string
	var probeAddr string
	targetLabels := []vulnerabilityreport.VulnerabilityLabel{}
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.IntVar(&maxJitterPercent, "max-jitter-percent", 10,
		"Spreads out re-queue interval of reports by +/- this amount to spread load.")

	flag.StringVar(&podIPString, "pod-ip", "", "The IP address of the current Pod/instance used when sharding reports.")

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

	setupLog.Info(fmt.Sprintf("This is exporter instance %s", podIP.String()))

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

	// Set up informer for our own service endpoints.
	resourceType := "Endpoints.v1.api"
	serviceName := "starboard-exporter"

	dc, err := dynamic.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		setupLog.Error(err, "unable to set up informer")
		os.Exit(1)
	}
	setupLog.Info("1")
	listOptionsFunc := dynamicinformer.TweakListOptionsFunc(func(options *metav1.ListOptions) {
		options.FieldSelector = "metadata.name=" + serviceName
	})
	// factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, 0, corev1.NamespaceAll, nil)
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, 0, corev1.NamespaceAll, listOptionsFunc)

	sgvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}
	gvr, g2 := schema.ParseResourceArg(resourceType)
	setupLog.Info(fmt.Sprintf("gvr: %v / g2: %v", gvr, g2))
	setupLog.Info("2")
	// informer := factory.ForResource(*gvr)
	informer := factory.ForResource(sgvr)
	inf := informer.Informer()

	stopper := make(chan struct{})
	defer close(stopper)
	setupLog.Info("3")
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("add")
			ep := &corev1.Endpoints{}
			fmt.Println(fmt.Sprintf("type: %s", reflect.TypeOf(obj)))
			fmt.Println(obj)

			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), ep)
			if err != nil {
				fmt.Println("could not convert obj to Endpoints")
				fmt.Print(err)
				return
			}
			fmt.Println(ep)

			// endp, ok := obj.(corev1.Endpoints)
			// if !ok {
			// 	fmt.Println("could not convert obj to Endpoints")
			// 	fmt.Println(err)
			// 	return
			// }
			// fmt.Println(endp)
			// try following https://erwinvaneyk.nl/kubernetes-unstructured-to-typed/
			fmt.Println(ep.Subsets)
			// err := runtime.DefaultUnstructuredConverter.
			// 	FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), ep)
			// if err != nil {
			// 	fmt.Println("could not convert obj to Endpoints")
			// 	fmt.Print(err)
			// 	return
			// }
			fmt.Println(fmt.Sprintf("found ep: %v", ep))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			ep := &corev1.Endpoints{}

			err := runtime.DefaultUnstructuredConverter.FromUnstructured(newObj.(*unstructured.Unstructured).UnstructuredContent(), ep)
			if err != nil {
				fmt.Println("could not convert obj to Endpoints")
				fmt.Print(err)
				return
			}
			fmt.Println(fmt.Sprintf("updated ep: %v", ep))
		},
	}
	setupLog.Info("4")
	inf.AddEventHandler(handlers)
	setupLog.Info("5")
	inf.Run(stopper)
	setupLog.Info("6")
	// factory := informers.NewSharedInformerFactory(mgr.GetClient(), 5*time.Minute)

	// Set up consistent hashing to shard reports over all of our exporters.

	if err = (&vulnerabilityreport.VulnerabilityReportReconciler{
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
