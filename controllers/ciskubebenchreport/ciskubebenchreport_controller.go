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
	ShardHelper      *utils.ShardHelper
	TargetLabels     []ReportLabel
}

//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport/finalizers,verbs=update

func (r *CISKubeBenchReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	_ = r.Log.WithValues("ciskubebenchreport", req.NamespacedName)

	registerMetricsOnce.Do(r.registerMetrics)

	// The report has changed, meaning our metrics are out of date for this report. Clear them.
	deletedSummaries := BenchmarkSummary.DeletePartialMatch(prometheus.Labels{"node_name": req.Name})
	deletedSections := BenchmarkSectionSummary.DeletePartialMatch(prometheus.Labels{"node_name": req.Name})
	deletedResults := BenchmarkResultInfo.DeletePartialMatch(prometheus.Labels{"node_name": req.Name})

	shouldOwn := r.ShardHelper.ShouldOwn(req.Name)
	if shouldOwn {

		// Try to get the report. It might not exist anymore, in which case we don't need to do anything.
		report := &aqua.CISKubeBenchReport{}
		if err := r.Client.Get(ctx, req.NamespacedName, report); err != nil {
			if apierrors.IsNotFound(err) {
				// Most likely the report was deleted.
				return ctrl.Result{}, nil
			}

			// Error reading the object.
			r.Log.Error(err, "Unable to read report")
			return ctrl.Result{}, err
		}

		r.Log.Info(fmt.Sprintf("Reconciled %s || Found (P/I/W/F): %d/%d/%d/%d",
			req.Name,
			report.Report.Summary.PassCount,
			report.Report.Summary.InfoCount,
			report.Report.Summary.WarnCount,
			report.Report.Summary.FailCount,
		))

		r.publishCISMetrics(report)

		if utils.SliceContains(report.GetFinalizers(), CISKubeBenchReportFinalizer) {
			// Remove the finalizer if we're the shard owner.
			ctrlutil.RemoveFinalizer(report, CISKubeBenchReportFinalizer)
			if err := r.Update(ctx, report); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Add a label to this report so any previous owners will reconcile and drop the metric.
		report.Labels[controllers.ShardOwnerLabel] = r.ShardHelper.PodIP
		err := r.Client.Update(ctx, report, &client.UpdateOptions{})
		if err != nil {
			r.Log.Error(err, "unable to add shard owner label")
		}
	} else {
		if deletedSummaries > 0 || deletedSections > 0 || deletedResults > 0 {
			r.Log.Info(fmt.Sprintf("cleared %d report summary, %d section summary, and %d detail metrics", deletedSummaries, deletedSections, deletedResults))
		}
	}

	return utils.JitterRequeue(controllers.DefaultRequeueDuration, r.MaxJitterPercent, r.Log), nil
}

func (r *CISKubeBenchReportReconciler) publishCISMetrics(report *aqua.CISKubeBenchReport) {
	// Publish summary metrics.
	publishSummaryMetrics(report)

	// Publish section metrics
	publishSectionMetrics(report)

	// If we have custom metrics to expose, do it.
	if len(r.TargetLabels) > 0 {
		publishResultMetrics(report, r.TargetLabels)
	}
}

func RequeueReportsForPod(c client.Client, log logr.Logger, podIP string) {
	cisList := &aqua.CISKubeBenchReportList{}
	opts := []client.ListOption{
		client.MatchingLabels{controllers.ShardOwnerLabel: podIP},
	}

	// Get the list of reports with our label.
	err := c.List(context.Background(), cisList, opts...)
	if err != nil {
		log.Error(err, "unable to list ciskubebenchreports")
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

	BenchmarkResultInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "result_info",
			Help:      "Exposes the detailed information of a test",
		},
		LabelNamesForList(r.TargetLabels),
	)

	metrics.Registry.MustRegister(BenchmarkResultInfo)
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

func getCountPerResult(report *aqua.CISKubeBenchReport) map[string]float64 {
	return map[string]float64{
		"PASS": float64(report.Report.Summary.PassCount),
		"INFO": float64(report.Report.Summary.InfoCount),
		"WARN": float64(report.Report.Summary.WarnCount),
		"FAIL": float64(report.Report.Summary.FailCount),
	}
}

func getCountPerResultSection(section aqua.CISKubeBenchSection) map[string]float64 {
	return map[string]float64{
		"PASS": float64(section.TotalPass),
		"INFO": float64(section.TotalInfo),
		"WARN": float64(section.TotalWarn),
		"FAIL": float64(section.TotalFail),
	}
}

func publishSummaryMetrics(report *aqua.CISKubeBenchReport) {
	summaryValues := valuesForReport(report, LabelsForGroup(labelGroupSummary))

	for status, count := range getCountPerResult(report) {
		// Expose the metric.
		summaryValues["status"] = status
		BenchmarkSummary.With(
			summaryValues,
		).Set(count)
	}

}

func publishSectionMetrics(report *aqua.CISKubeBenchReport) {

	reportValues := valuesForReport(report, LabelsForGroup(labelGroupSummary))

	// Add node name to section metrics
	for _, s := range report.Report.Sections {

		for status, count := range getCountPerResultSection(s) {
			secValues := valuesForSection(s, LabelsForGroup(labelGroupSectionSummary))

			secValues["node_name"] = reportValues["node_name"]
			secValues["status"] = status

			//Expose the metric.
			BenchmarkSectionSummary.With(
				secValues,
			).Set(count)
		}

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

				// Inherit labels from report and section only if they are enabled
				if _, found := reportValues["node_name"]; found {
					resValues["node_name"] = reportValues["node_name"]
				}
				if _, found := secValues["node_type"]; found {
					resValues["node_type"] = secValues["node_type"]
				}
				if _, found := secValues["section_name"]; found {
					resValues["section_name"] = secValues["section_name"]
				}

				//Expose the metric.
				BenchmarkResultInfo.With(
					resValues,
				).Set(1)
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
	case "node_type":
		return sec.NodeType
	case "section_name":
		return sec.Text
	default:
		// Error?
		return ""
	}
}

func reportValueFor(field string, report *aqua.CISKubeBenchReport) string {
	switch field {
	case "node_name":
		return report.Name
	default:
		// Error?
		return ""
	}
}
