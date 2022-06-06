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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

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
}

//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aquasecurity.github.io.giantswarm,resources=ciskubebenchreport/finalizers,verbs=update

func (r *CISKubeBenchReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	_ = r.Log.WithValues("ciskubebenchreport", req.NamespacedName)

	registerMetricsOnce.Do(r.registerMetrics)

	report := &aqua.CISKubeBenchReport{}
	if err := r.Client.Get(ctx, req.NamespacedName, report); err != nil {
		if apierrors.IsNotFound(err) {
			// Most likely the report was deleted.
			return ctrl.Result{}, nil
		}

		// Error reading the object.
		r.Log.Error(err, "Unable to read CISKubeBenchReport")
		return ctrl.Result{}, err
	}
	r.Log.Info(fmt.Sprintf("Reconciled %s",
		report.Report.Summary,
	))
	// spew.Dump(report)
	/*
		if report.DeletionTimestamp.IsZero() {
			// Give the report our finalizer if it doesn't have one.
			if !utils.SliceContains(report.GetFinalizers(), CISKubeBenchReportFinalizer) {
				ctrlutil.AddFinalizer(report, CISKubeBenchReportFinalizer)
				if err := r.Update(ctx, report); err != nil {
					return ctrl.Result{}, err
				}
			}

			r.Log.Info(fmt.Sprintf("Reconciled %s || Found (C/H/M/L): %d/%d/%d/%d",
				req.NamespacedName,
				report.Report.Summary.PassCount,
				report.Report.Summary.InfoCount,
				report.Report.Summary.WarnCount,
				report.Report.Summary.FailCount,
			))

			// Publish summary metrics for this report.
			publishSummaryMetrics(report)

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
	*/
	return utils.JitterRequeue(controllers.DefaultRequeueDuration, r.MaxJitterPercent, r.Log), nil
}

/*
func (r *CISKubeBenchReportReconciler) registerMetrics() {

	CISBenchmarkInfo = prometheus.NewGaugeVec(
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
*/
func (r *CISKubeBenchReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&aqua.CISKubeBenchReport{}).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed setting up controller with controller manager")
	}

	return nil
}

/*
func (r *CISKubeBenchReportReconciler) clearImageMetrics(report *aqua.CISKubeBenchReport) {
	// clear summary metrics
	summaryValues := valuesForReport(report, metricLabels)

	// Delete the series for each severity.
	for severity := range getCountPerSeverity(report) {
		v := summaryValues
		v["severity"] = severity

		// Delete the metric.
		CISKubeBenchSummary.Delete(
			v,
		)
	}
}

func getCountPerSeverity(report *aqua.ciskubebenchreportummary) map[string]float64 {
	// Format is e.g. {FAIL: 10}.
	return map[string]float64{
		string(aqua.ResultPass): float64(report.Report.Summary.PassCount),
		string(aqua.ResultInfo): float64(report.Report.Summary.InfoCount),
		string(aqua.ResultWarn): float64(report.Report.Summary.WarnCount),
		string(aqua.ResultFail): float64(report.Report.Summary.FailCount),
	}
}

func publishSummaryMetrics(report *aqua.CISKubeBenchReport) {
	summaryValues := valuesForReport(report, metricLabels)

	// Add the severity label after the standard labels and expose each severity metric.
	for severity, count := range getCountPerSeverity(report) {
		v := summaryValues
		v["severity"] = severity

		// Expose the metric.
		CISKubeBenchSummary.With(
			v,
		).Set(count)
	}
}

func valuesForReport(report *aqua.CISKubeBenchReport, labels []string) map[string]string {
	result := map[string]string{}
	for _, label := range labels {
		result[label] = reportValueFor(label, report)
	}
	return result
}

func reportValueFor(field string, report *aqua.CISKubeBenchReport) string {
	switch field {
	case "resource_name":
		return report.Name
	case "resource_namespace":
		return report.Namespace
	case "severity":
		return "" // this value will be overwritten on publishSummaryMetrics
	default:
		// Error?
		return ""
	}
}
*/
