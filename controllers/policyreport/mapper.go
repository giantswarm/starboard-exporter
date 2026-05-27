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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	kubescape "github.com/kubescape/storage/pkg/apis/softwarecomposition/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	policyreport "sigs.k8s.io/wg-policy-prototypes/policy-report/apis/wgpolicyk8s.io/v1alpha2"
)

const (
	// Source identifies these results in the Policy Reporter UI. The UI plugin filters on this exact string.
	policyReportSource   = "Kubescape Vulnerability"
	policyReportCategory = "Vulnerability Scan"

	// Prefer the CVSS 3.1 base score when several CVSS versions are present.
	preferredCvssVersion = "3.1"

	managedByLabel = "app.kubernetes.io/managed-by"
	managedByValue = "starboard-exporter"
	sourceLabel    = "policy-reporter/source"
	sourceValue    = "kubescape"

	// Well-known labels Kubescape sets on VulnerabilityManifestSummary objects to identify the scanned workload.
	workloadKindLabel       = "kubescape.io/workload-kind"
	workloadNameLabel       = "kubescape.io/workload-name"
	workloadNamespaceLabel  = "kubescape.io/workload-namespace"
	workloadAPIGroupLabel   = "kubescape.io/workload-api-group"
	workloadAPIVersionLabel = "kubescape.io/workload-api-version"

	// Annotation carrying the scanned image reference on the image-level VulnerabilityManifest.
	kubescapeImageTagAnnotation = "kubescape.io/image-tag"

	// maxNameLength is the maximum length of a Kubernetes object name (RFC 1123 DNS subdomain).
	maxNameLength = 253
	// nameHashLength is the length of the hash suffix appended to truncated names.
	nameHashLength = 10
)

// knownKinds maps Kubescape's (sometimes lowercase) workload kind to the canonical
// Kubernetes Kind so the Policy Reporter UI and kubectl display them correctly.
var knownKinds = map[string]string{
	"pod":                   "Pod",
	"replicaset":            "ReplicaSet",
	"replicationcontroller": "ReplicationController",
	"deployment":            "Deployment",
	"statefulset":           "StatefulSet",
	"daemonset":             "DaemonSet",
	"job":                   "Job",
	"cronjob":               "CronJob",
	"node":                  "Node",
}

// PolicyResult values. The wgpolicyk8s.io/v1alpha2 package does not export these as constants.
const (
	resultPass  = policyreport.PolicyResult("pass")
	resultFail  = policyreport.PolicyResult("fail")
	resultWarn  = policyreport.PolicyResult("warn")
	resultError = policyreport.PolicyResult("error")
	resultSkip  = policyreport.PolicyResult("skip")
)

// policyReportName derives the deterministic PolicyReport name from a summary name.
// It is a pure function of the summary name so the report can be located for deletion
// even after the summary is gone. If the name would exceed the Kubernetes limit it is
// truncated and a short hash of the full name is appended to keep it unique and stable.
func policyReportName(summaryName string) string {
	name := fmt.Sprintf("kubescape-%s", summaryName)
	if len(name) <= maxNameLength {
		return name
	}

	sum := sha256.Sum256([]byte(name))
	suffix := hex.EncodeToString(sum[:])[:nameHashLength]
	truncated := strings.TrimRight(name[:maxNameLength-nameHashLength-1], "-.")
	return truncated + "-" + suffix
}

// normalizeKind returns the canonical Kubernetes Kind for a (possibly lowercase) kind label.
func normalizeKind(kind string) string {
	if canonical, ok := knownKinds[strings.ToLower(kind)]; ok {
		return canonical
	}
	if kind == "" {
		return kind
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

// severityToResult maps a Kubescape/Grype severity to a PolicyReport result status.
// critical/high -> fail, medium -> warn, low/negligible -> skip, anything else (incl. unknown) -> error.
func severityToResult(severity string) policyreport.PolicyResult {
	switch strings.ToUpper(severity) {
	case "CRITICAL", "HIGH":
		return resultFail
	case "MEDIUM":
		return resultWarn
	case "LOW", "NEGLIGIBLE":
		return resultSkip
	default:
		return resultError
	}
}

// severityToSeverity maps a Kubescape/Grype severity to a PolicyReportResult severity.
// The v1alpha2 severity enum is critical;high;low;medium;info, so negligible/unknown collapse to info.
func severityToSeverity(severity string) policyreport.PolicyResultSeverity {
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		return policyreport.PolicyResultSeverity("critical")
	case "HIGH":
		return policyreport.PolicyResultSeverity("high")
	case "MEDIUM":
		return policyreport.PolicyResultSeverity("medium")
	case "LOW":
		return policyreport.PolicyResultSeverity("low")
	default:
		return policyreport.PolicyResultSeverity("info")
	}
}

// cvssScore returns the preferred CVSS base score for a match, favouring v3.1 then the highest version.
func cvssScore(match kubescape.Match) float64 {
	score := float64(0)
	highestVersion := ""
	for _, cvss := range match.Vulnerability.Cvss {
		if cvss.Version == preferredCvssVersion {
			return cvss.Metrics.BaseScore
		}
		if cvss.Version > highestVersion {
			highestVersion = cvss.Version
			score = cvss.Metrics.BaseScore
		}
	}
	return score
}

// workloadSubject builds an ObjectReference to the scanned workload from the summary's labels.
// Returns nil if the labels don't identify a workload.
func workloadSubject(summary *kubescape.VulnerabilityManifestSummary) *corev1.ObjectReference {
	labels := summary.Labels
	if labels == nil {
		return nil
	}

	kind := labels[workloadKindLabel]
	name := labels[workloadNameLabel]
	if kind == "" || name == "" {
		return nil
	}

	apiVersion := labels[workloadAPIVersionLabel]
	if group := labels[workloadAPIGroupLabel]; group != "" && apiVersion != "" {
		apiVersion = group + "/" + apiVersion
	}

	return &corev1.ObjectReference{
		Kind:       normalizeKind(kind),
		Name:       name,
		Namespace:  labels[workloadNamespaceLabel],
		APIVersion: apiVersion,
	}
}

func imageTag(manifest *kubescape.VulnerabilityManifest) string {
	if manifest.Annotations == nil {
		return ""
	}
	return manifest.Annotations[kubescapeImageTagAnnotation]
}

// matchToResult converts a single Grype match (one CVE on one package) into a PolicyReportResult.
func matchToResult(match kubescape.Match, subject *corev1.ObjectReference, image string, timestamp metav1.Timestamp) policyreport.PolicyReportResult {
	severity := match.Vulnerability.Severity

	properties := map[string]string{
		"resource":         match.Artifact.Name,
		"installedVersion": match.Artifact.Version,
	}
	if len(match.Vulnerability.Fix.Versions) > 0 {
		properties["fixedVersion"] = match.Vulnerability.Fix.Versions[0]
	}
	if score := cvssScore(match); score > 0 {
		properties["cvssScore"] = strconv.FormatFloat(score, 'f', -1, 64)
	}
	if len(match.Vulnerability.URLs) > 0 {
		properties["primaryLink"] = match.Vulnerability.URLs[0]
	}
	if match.Artifact.PURL != "" {
		properties["purl"] = match.Artifact.PURL
	}
	if image != "" {
		properties["image"] = image
	}

	message := match.Vulnerability.Description
	if message == "" {
		message = match.Vulnerability.ID
	}

	result := policyreport.PolicyReportResult{
		Source:      policyReportSource,
		Policy:      match.Vulnerability.ID,
		Rule:        match.Artifact.Name,
		Category:    policyReportCategory,
		Severity:    severityToSeverity(severity),
		Result:      severityToResult(severity),
		Scored:      true,
		Timestamp:   timestamp,
		Description: message,
		Properties:  properties,
	}
	if subject != nil {
		result.Subjects = []corev1.ObjectReference{*subject}
	}
	return result
}

// summaryToReport builds the desired PolicyReport for a workload from its summary and the
// referenced image-level manifest (which holds the per-CVE detail).
func summaryToReport(summary *kubescape.VulnerabilityManifestSummary, manifest *kubescape.VulnerabilityManifest) *policyreport.PolicyReport {
	subject := workloadSubject(summary)
	image := imageTag(manifest)
	timestamp := metav1.Timestamp{Seconds: manifest.CreationTimestamp.Unix()}

	results := make([]policyreport.PolicyReportResult, 0, len(manifest.Spec.Payload.Matches))
	counts := policyreport.PolicyReportSummary{}
	for _, match := range manifest.Spec.Payload.Matches {
		result := matchToResult(match, subject, image, timestamp)
		results = append(results, result)

		switch result.Result {
		case resultPass:
			counts.Pass++
		case resultFail:
			counts.Fail++
		case resultWarn:
			counts.Warn++
		case resultSkip:
			counts.Skip++
		case resultError:
			counts.Error++
		}
	}

	return &policyreport.PolicyReport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyReportName(summary.Name),
			Namespace: summary.Namespace,
			Labels: map[string]string{
				managedByLabel: managedByValue,
				sourceLabel:    sourceValue,
			},
		},
		Scope:   subject,
		Summary: counts,
		Results: results,
	}
}
