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

package ciskubebenchreport

import (
	"context"
	"fmt"
	"sync"

	aqua "github.com/aquasecurity/starboard/pkg/apis/aquasecurity/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/giantswarm/starboard-exporter/controllers"
	"github.com/giantswarm/starboard-exporter/utils"
)

const (
	CISKubeBenchReportFinalizer = "starboard-exporter.giantswarm.io/ciskubebenchreport"
)

var registerMetricsOnce sync.Once

// CISKubeBenchReportReconciler reconciles a CISKubeBenchReport object
type CISKubeBenchReportReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	MaxJitterPercent int
	TargetLabels     []ReportLabel
}

//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport/finalizers,verbs=update

func (r *CISKubeBenchReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	_ = r.Log.WithValues("ciskubebenchreport", req.NamespacedName)

	registerMetricsOnce.Do(r.registerMetrics)

	report := &aqua.CISKubeBenchReport{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, report); err != nil {
		if apierrors.IsNotFound(err) {
			// Most likely the report was deleted.
			return ctrl.Result{}, nil
		}

		// Error reading the object.
		r.Log.Error(err, "Unable to read CISKubeBenchReport")
		return ctrl.Result{}, err
	}

	if report.DeletionTimestamp.IsZero() {
		// Give the report our finalizer if it doesn't have one.
		if !utils.SliceContains(report.GetFinalizers(), CISKubeBenchReportFinalizer) {
			ctrlutil.AddFinalizer(report, CISKubeBenchReportFinalizer)
			if err := r.Update(ctx, report); err != nil {
				return ctrl.Result{}, err
			}
		}

		r.Log.Info(fmt.Sprintf("Reconciled %s || Found (P/I/W/F): %d/%d/%d/%d",
			req.NamespacedName,
			report.Report.Summary.PassCount,
			report.Report.Summary.InfoCount,
			report.Report.Summary.WarnCount,
			report.Report.Summary.FailCount,
		))

		// Publish summary metrics for this report.
		//publishSummaryMetrics(report)
		//publishSectionMetrics(report, r.TargetLabels)
		//publishResultMetrics(report, r.TargetLabels)

	} else {

		if utils.SliceContains(report.GetFinalizers(), CISKubeBenchReportFinalizer) {
			// Unfortunately, we can't just clear the series based on one label value,
			// we have to reconstruct all of the label values to delete the series.
			// That's the only reason the finalizer is needed at all.
			r.clearImageMetrics(report)

			ctrlutil.RemoveFinalizer(report, CISKubeBenchReportFinalizer)
			if err := r.Update(ctx, report); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return utils.JitterRequeue(controllers.DefaultRequeueDuration, r.MaxJitterPercent, r.Log), nil
}

func RequeueReportsForPod(c client.Client, log logr.Logger, podIP string) {
	cisList := &aqua.CISKubeBenchReportList{}
	opts := []client.ListOption{
		client.MatchingLabels{controllers.ShardOwnerLabel: podIP},
	}

	// Get the list of reports with our label.
	err := c.List(context.Background(), cisList, opts...)
	if err != nil {
		log.Error(err, "unable to fetch ciskubebenchreport")
	}

	for _, r := range cisList.Items {
		// Retrieve the individual report.
		report := &aqua.CISKubeBenchReport{}
		err := c.Get(context.Background(), client.ObjectKey{Name: r.Name, Namespace: r.Namespace}, report)
		if err != nil {
			log.Error(err, "unable to fetch ciskubebenchreport")
		}

		// Clear the shard-owner label if it still has our label
		if r.Labels[controllers.ShardOwnerLabel] == podIP {
			r.Labels[controllers.ShardOwnerLabel] = ""
			err = c.Update(context.Background(), report, &client.UpdateOptions{})
			if err != nil {
				log.Error(err, fmt.Sprintf("unable to remove %s label", controllers.ShardOwnerLabel))
			}
		}
	}
}

func (r *CISKubeBenchReportReconciler) registerMetrics() {

	CISBenchmarkInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "cis_benchmarks",
			Help:      "Indicates the results of a CIS benchmark.",
		},
		LabelNamesForList(r.TargetLabels),
	)

	metrics.Registry.MustRegister(CISBenchmarkInfo)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CISKubeBenchReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&aqua.CISKubeBenchReport{}).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed setting up controller with controller manager")
	}

	return nil
}

func (r *CISKubeBenchReportReconciler) clearImageMetrics(report *aqua.CISKubeBenchReport) {
	// clear summary metrics
	summaryValues := valuesForReport(report, LabelsForGroup(labelGroupSummary))

	// Delete the series for each node.
	for severity := range getCountPerResult(report) {
		v := summaryValues
		v["severity"] = severity

		// Delete the metric.
		BenchmarkSummary.Delete(
			v,
		)
	}
}

func getCountPerResult(report *aqua.CISKubeBenchReport) map[string]float64 {
	return map[string]float64{
		"PassCount": float64(report.Report.Summary.PassCount),
		"InfoCount": float64(report.Report.Summary.InfoCount),
		"WarnCount": float64(report.Report.Summary.WarnCount),
		"FailCount": float64(report.Report.Summary.FailCount),
	}
}

func publishSummaryMetrics(report *aqua.CISKubeBenchReport) {
	summaryValues := valuesForReport(report, LabelsForGroup(labelGroupSummary))

	v := summaryValues

	// Expose the metric.
	BenchmarkSummary.With(
		v,
	)

}

func publishSectionMetrics(report *aqua.CISKubeBenchReport, targetLabels []ReportLabel) {

	reportValues := valuesForReport(report, targetLabels)

	// Add node name to section metrics
	for _, s := range report.Report.Sections {
		secValues := valuesForSection(s, targetLabels)

		secValues["node_name"] = reportValues["node_name"]

		//Expose the metric.
		BenchmarkSectionSummary.With(
			secValues,
		)
	}
}

func publishResultMetrics(report *aqua.CISKubeBenchReport, targetLabels []ReportLabel) {

	reportValues := valuesForReport(report, targetLabels)

	// Add node name to section metrics
	for _, s := range report.Report.Sections {
		secValues := valuesForSection(s, targetLabels)

		for _, t := range s.Tests {
			// Add node name and node type to result metrics
			for _, r := range t.Results {
				resValues := valuesForResult(r, targetLabels)

				resValues["node_name"] = reportValues["node_name"]
				resValues["node_type"] = secValues["node_type"]

				//Expose the metric.
				BenchmarkTestInfo.With(
					resValues,
				)
			}
		}
	}

}

func valuesForReport(report *aqua.CISKubeBenchReport, labels []ReportLabel) map[string]string {
	result := map[string]string{}
	for _, label := range labels {
		if label.Scope == FieldScopeReport {
			result[label.Name] = reportValueFor(label.Name, report)
		}
	}
	return result
}

func valuesForSection(sec aqua.CISKubeBenchSection, labels []ReportLabel) map[string]string {
	result := map[string]string{}
	for _, label := range labels {
		if label.Scope == FieldScopeSection {
			result[label.Name] = secValueFor(label.Name, sec)
		}
	}
	return result
}

func valuesForResult(res aqua.CISKubeBenchResult, labels []ReportLabel) map[string]string {
	result := map[string]string{}
	for _, label := range labels {
		if label.Scope == FieldScopeResult {
			result[label.Name] = resValueFor(label.Name, res)
		}
	}
	return result
}

func resValueFor(field string, res aqua.CISKubeBenchResult) string {
	switch field {
	case "test_number":
		return res.TestNumber
	case "test_desc":
		return res.TestDesc
	case "test_status":
		return res.Status
	default:
		// Error?
		return ""
	}
}

func secValueFor(field string, sec aqua.CISKubeBenchSection) string {
	switch field {
	case "total_fail":
		return fmt.Sprint(sec.TotalFail)
	case "total_pass":
		return fmt.Sprint(sec.TotalPass)
	case "total_info":
		return fmt.Sprint(sec.TotalWarn)
	case "total_warn":
		return fmt.Sprint(sec.TotalWarn)
	case "node_type":
		return sec.NodeType
	default:
		// Error?
		return ""
	}
}

func reportValueFor(field string, report *aqua.CISKubeBenchReport) string {
	switch field {
	case "node_name":
		return report.Name
	case "fail_count":
		return fmt.Sprint(report.Report.Summary.FailCount)
	case "pass_count":
		return fmt.Sprint(report.Report.Summary.PassCount)
	case "info_count":
		return fmt.Sprint(report.Report.Summary.InfoCount)
	case "warn_count":
		return fmt.Sprint(report.Report.Summary.WarnCount)
	default:
		// Error?
		return ""
	}
}
