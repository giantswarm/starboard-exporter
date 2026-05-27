/*
	Copyright 2025 Giant Swarm.

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

package policyreport

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kubescape "github.com/kubescape/storage/pkg/apis/softwarecomposition/v1beta1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	policyreport "sigs.k8s.io/wg-policy-prototypes/policy-report/apis/wgpolicyk8s.io/v1alpha2"

	"github.com/giantswarm/starboard-exporter/controllers"
	"github.com/giantswarm/starboard-exporter/utils"
)

// KubescapePolicyReportReconciler turns Kubescape VulnerabilityManifestSummary objects into
// wgpolicyk8s.io PolicyReports so vulnerability findings appear in the Policy Reporter UI.
type KubescapePolicyReportReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	MaxJitterPercent int
	ShardHelper      *utils.ShardHelper

	// KubescapeNamespace is where the Kubescape storage component keeps image-level
	// VulnerabilityManifests. The summary's ref records the workload namespace instead,
	// so we resolve manifests here first.
	KubescapeNamespace string
}

func (r *KubescapePolicyReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := r.Log.WithValues("vulnerabilitymanifestsummary", req.NamespacedName)

	// Only the shard owner writes the report, so replicas don't race to write the same object.
	if !r.ShardHelper.ShouldOwn(req.String()) {
		return utils.JitterRequeue(controllers.DefaultRequeueDuration, r.MaxJitterPercent, r.Log), nil
	}

	summary := &kubescape.VulnerabilityManifestSummary{}
	if err := r.Get(ctx, req.NamespacedName, summary); err != nil {
		if apierrors.IsNotFound(err) {
			// The summary is gone; remove the derived report. The owner reference handles this
			// too when garbage collection runs, but we delete explicitly to converge immediately.
			return ctrl.Result{}, r.deleteReport(ctx, req.NamespacedName)
		}
		logger.Error(err, "unable to read VulnerabilityManifestSummary")
		return ctrl.Result{}, err
	}

	// The summary references the image-level manifest that holds the per-CVE detail (all vulnerabilities).
	ref := summary.Spec.Vulnerabilities.ImageVulnerabilitiesObj
	if ref.Name == "" {
		logger.Info("summary has no image vulnerabilities reference, skipping")
		return utils.JitterRequeue(controllers.DefaultRequeueDuration, r.MaxJitterPercent, r.Log), nil
	}

	manifest, err := r.resolveManifest(ctx, ref)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The manifest may not have been written yet; requeue and try again later.
			logger.Info("referenced VulnerabilityManifest not found, requeueing", "ref", ref.Name)
			return utils.JitterRequeue(controllers.DefaultRequeueDuration, r.MaxJitterPercent, r.Log), nil
		}
		logger.Error(err, "unable to read referenced VulnerabilityManifest", "ref", ref.Name)
		return ctrl.Result{}, err
	}

	desired := summaryToReport(summary, manifest)
	if err := r.upsertReport(ctx, summary, desired); err != nil {
		logger.Error(err, "unable to upsert PolicyReport")
		return ctrl.Result{}, err
	}

	logger.Info(fmt.Sprintf("reconciled PolicyReport %s/%s with %d results", desired.Namespace, desired.Name, len(desired.Results)))
	return utils.JitterRequeue(controllers.DefaultRequeueDuration, r.MaxJitterPercent, r.Log), nil
}

// resolveManifest fetches the image-level VulnerabilityManifest referenced by a summary.
// Kubescape stores image manifests in its storage namespace, but the summary's ref records
// the workload namespace, so we look in the Kubescape namespace first and fall back to the
// ref's namespace for installs where the two coincide.
func (r *KubescapePolicyReportReconciler) resolveManifest(ctx context.Context, ref kubescape.VulnerabilitiesObjScope) (*kubescape.VulnerabilityManifest, error) {
	candidates := []string{r.KubescapeNamespace}
	if ref.Namespace != "" && ref.Namespace != r.KubescapeNamespace {
		candidates = append(candidates, ref.Namespace)
	}

	manifest := &kubescape.VulnerabilityManifest{}
	var err error
	for _, ns := range candidates {
		if ns == "" {
			continue
		}
		err = r.Get(ctx, client.ObjectKey{Namespace: ns, Name: ref.Name}, manifest)
		if err == nil {
			return manifest, nil
		}
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	}
	return nil, err
}

func (r *KubescapePolicyReportReconciler) upsertReport(ctx context.Context, summary *kubescape.VulnerabilityManifestSummary, desired *policyreport.PolicyReport) error {
	existing := &policyreport.PolicyReport{}
	err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if apierrors.IsNotFound(err) {
		// Owner reference ties the report's lifecycle to the summary (same namespace) for GC.
		if err := ctrl.SetControllerReference(summary, desired, r.Scheme); err != nil {
			return errors.Wrap(err, "failed setting owner reference")
		}
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	existing.Scope = desired.Scope
	existing.Summary = desired.Summary
	existing.Results = desired.Results
	if existing.Labels == nil {
		existing.Labels = map[string]string{}
	}
	for k, v := range desired.Labels {
		existing.Labels[k] = v
	}
	return r.Update(ctx, existing)
}

func (r *KubescapePolicyReportReconciler) deleteReport(ctx context.Context, summaryKey apitypes.NamespacedName) error {
	report := &policyreport.PolicyReport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyReportName(summaryKey.Name),
			Namespace: summaryKey.Namespace,
		},
	}
	if err := r.Delete(ctx, report); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubescapePolicyReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&kubescape.VulnerabilityManifestSummary{}).
		Owns(&policyreport.PolicyReport{}).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed setting up policyreport controller with controller manager")
	}
	return nil
}
