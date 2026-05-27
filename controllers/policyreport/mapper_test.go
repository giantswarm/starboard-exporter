package policyreport

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	kubescape "github.com/kubescape/storage/pkg/apis/softwarecomposition/v1beta1"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	policyreport "sigs.k8s.io/wg-policy-prototypes/policy-report/apis/wgpolicyk8s.io/v1alpha2"
)

func Test_severityToResult(t *testing.T) {
	testCases := []struct {
		severity string
		expected policyreport.PolicyResult
	}{
		{"Critical", resultFail},
		{"critical", resultFail},
		{"High", resultFail},
		{"Medium", resultWarn},
		{"Low", resultSkip},
		{"Negligible", resultSkip},
		{"Unknown", resultError},
		{"", resultError},
		{"something-weird", resultError},
	}

	for _, tc := range testCases {
		t.Run(tc.severity, func(t *testing.T) {
			assert.Equal(t, tc.expected, severityToResult(tc.severity))
		})
	}
}

func Test_severityToSeverity(t *testing.T) {
	testCases := []struct {
		severity string
		expected policyreport.PolicyResultSeverity
	}{
		{"Critical", policyreport.PolicyResultSeverity("critical")},
		{"High", policyreport.PolicyResultSeverity("high")},
		{"Medium", policyreport.PolicyResultSeverity("medium")},
		{"Low", policyreport.PolicyResultSeverity("low")},
		{"Negligible", policyreport.PolicyResultSeverity("info")},
		{"Unknown", policyreport.PolicyResultSeverity("info")},
		{"", policyreport.PolicyResultSeverity("info")},
	}

	for _, tc := range testCases {
		t.Run(tc.severity, func(t *testing.T) {
			assert.Equal(t, tc.expected, severityToSeverity(tc.severity))
		})
	}
}

func Test_policyReportName(t *testing.T) {
	assert.Equal(t, "kubescape-kubescape-deployment-storage-apiserver", policyReportName("kubescape-deployment-storage-apiserver"))
}

func Test_policyReportName_truncatesLongNames(t *testing.T) {
	long := strings.Repeat("a", 300)
	name := policyReportName(long)

	assert.Assert(t, len(name) <= maxNameLength, "name length %d exceeds limit", len(name))
	assert.Assert(t, strings.HasPrefix(name, "kubescape-a"))
	// Must not end on a separator (would be an invalid Kubernetes name).
	assert.Assert(t, !strings.HasSuffix(name, "-") && !strings.HasSuffix(name, "."))
	// Deterministic for the same input.
	assert.Equal(t, name, policyReportName(long))
}

func Test_normalizeKind(t *testing.T) {
	testCases := []struct {
		in       string
		expected string
	}{
		{"deployment", "Deployment"},
		{"Deployment", "Deployment"},
		{"replicaset", "ReplicaSet"},
		{"daemonset", "DaemonSet"},
		{"statefulset", "StatefulSet"},
		{"cronjob", "CronJob"},
		{"job", "Job"},
		{"pod", "Pod"},
		{"replicationcontroller", "ReplicationController"},
		{"widget", "Widget"},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeKind(tc.in))
		})
	}
}

func Test_cvssScore(t *testing.T) {
	testCases := []struct {
		name     string
		cvss     []kubescape.Cvss
		expected float64
	}{
		{
			name:     "no cvss",
			cvss:     nil,
			expected: 0,
		},
		{
			name: "prefers 3.1 over other versions",
			cvss: []kubescape.Cvss{
				{Version: "2.0", Metrics: kubescape.CvssMetrics{BaseScore: 5.0}},
				{Version: "3.1", Metrics: kubescape.CvssMetrics{BaseScore: 9.8}},
				{Version: "3.0", Metrics: kubescape.CvssMetrics{BaseScore: 8.0}},
			},
			expected: 9.8,
		},
		{
			name: "falls back to highest version when no 3.1",
			cvss: []kubescape.Cvss{
				{Version: "2.0", Metrics: kubescape.CvssMetrics{BaseScore: 5.0}},
				{Version: "3.0", Metrics: kubescape.CvssMetrics{BaseScore: 8.0}},
			},
			expected: 8.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := kubescape.Match{
				Vulnerability: kubescape.Vulnerability{
					VulnerabilityMetadata: kubescape.VulnerabilityMetadata{Cvss: tc.cvss},
				},
			}
			assert.Equal(t, tc.expected, cvssScore(match))
		})
	}
}

func Test_workloadSubject(t *testing.T) {
	testCases := []struct {
		name     string
		labels   map[string]string
		expected *corev1.ObjectReference
	}{
		{
			name:     "no labels",
			labels:   nil,
			expected: nil,
		},
		{
			name:     "missing kind",
			labels:   map[string]string{workloadNameLabel: "storage"},
			expected: nil,
		},
		{
			name: "lowercase kind is normalized to canonical Kind",
			labels: map[string]string{
				workloadKindLabel:       "deployment",
				workloadNameLabel:       "storage",
				workloadNamespaceLabel:  "kubescape",
				workloadAPIGroupLabel:   "apps",
				workloadAPIVersionLabel: "v1",
			},
			expected: &corev1.ObjectReference{
				Kind:       "Deployment",
				Name:       "storage",
				Namespace:  "kubescape",
				APIVersion: "apps/v1",
			},
		},
		{
			name: "core group has no api group prefix",
			labels: map[string]string{
				workloadKindLabel:       "pod",
				workloadNameLabel:       "mypod",
				workloadNamespaceLabel:  "default",
				workloadAPIVersionLabel: "v1",
			},
			expected: &corev1.ObjectReference{
				Kind:       "Pod",
				Name:       "mypod",
				Namespace:  "default",
				APIVersion: "v1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summary := &kubescape.VulnerabilityManifestSummary{
				ObjectMeta: metav1.ObjectMeta{Labels: tc.labels},
			}
			if diff := cmp.Diff(tc.expected, workloadSubject(summary)); diff != "" {
				t.Fatalf("unexpected subject (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_matchToResult(t *testing.T) {
	subject := &corev1.ObjectReference{Kind: "Deployment", Name: "storage", Namespace: "kubescape", APIVersion: "apps/v1"}
	timestamp := metav1.Timestamp{Seconds: 1700000000}

	match := kubescape.Match{
		Vulnerability: kubescape.Vulnerability{
			VulnerabilityMetadata: kubescape.VulnerabilityMetadata{
				ID:          "CVE-2024-1234",
				Severity:    "High",
				Description: "a serious problem",
				URLs:        []string{"https://nvd.example/CVE-2024-1234", "https://other"},
				Cvss:        []kubescape.Cvss{{Version: "3.1", Metrics: kubescape.CvssMetrics{BaseScore: 7.5}}},
			},
			Fix: kubescape.Fix{Versions: []string{"1.2.4", "1.3.0"}},
		},
		Artifact: kubescape.GrypePackage{
			Name:    "openssl",
			Version: "1.2.3",
			PURL:    "pkg:deb/openssl@1.2.3",
		},
	}

	expected := policyreport.PolicyReportResult{
		Source:      "Kubescape Vulnerability",
		Policy:      "CVE-2024-1234",
		Rule:        "openssl",
		Category:    "Vulnerability Scan",
		Severity:    policyreport.PolicyResultSeverity("high"),
		Result:      resultFail,
		Scored:      true,
		Timestamp:   timestamp,
		Description: "a serious problem",
		Properties: map[string]string{
			"resource":         "openssl",
			"installedVersion": "1.2.3",
			"fixedVersion":     "1.2.4",
			"cvssScore":        "7.5",
			"primaryLink":      "https://nvd.example/CVE-2024-1234",
			"purl":             "pkg:deb/openssl@1.2.3",
			"image":            "quay.io/kubescape/storage:v1",
		},
		Subjects: []corev1.ObjectReference{*subject},
	}

	got := matchToResult(match, subject, "quay.io/kubescape/storage:v1", timestamp)
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("unexpected result (-want +got):\n%s", diff)
	}
}

func Test_matchToResult_fallsBackToIDWhenNoDescription(t *testing.T) {
	match := kubescape.Match{
		Vulnerability: kubescape.Vulnerability{
			VulnerabilityMetadata: kubescape.VulnerabilityMetadata{ID: "CVE-2024-1", Severity: "Low"},
		},
		Artifact: kubescape.GrypePackage{Name: "zlib", Version: "1.0"},
	}

	got := matchToResult(match, nil, "", metav1.Timestamp{})
	assert.Equal(t, "CVE-2024-1", got.Description)
	assert.Equal(t, resultSkip, got.Result)
	assert.Equal(t, 0, len(got.Subjects))
	// No fix, no cvss, no link, no purl, no image -> only resource + installedVersion.
	assert.Equal(t, 2, len(got.Properties))
}

func Test_summaryToReport(t *testing.T) {
	summary := &kubescape.VulnerabilityManifestSummary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubescape-deployment-storage-apiserver",
			Namespace: "kubescape",
			Labels: map[string]string{
				workloadKindLabel:       "deployment",
				workloadNameLabel:       "storage",
				workloadNamespaceLabel:  "kubescape",
				workloadAPIGroupLabel:   "apps",
				workloadAPIVersionLabel: "v1",
			},
		},
	}

	manifest := &kubescape.VulnerabilityManifest{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:       map[string]string{kubescapeImageTagAnnotation: "quay.io/kubescape/storage:v1"},
			CreationTimestamp: metav1.Unix(1700000000, 0),
		},
		Spec: kubescape.VulnerabilityManifestSpec{
			Payload: kubescape.GrypeDocument{
				Matches: []kubescape.Match{
					newMatch("CVE-1", "Critical", "pkgA"),
					newMatch("CVE-2", "High", "pkgB"),
					newMatch("CVE-3", "Medium", "pkgC"),
					newMatch("CVE-4", "Low", "pkgD"),
					newMatch("CVE-5", "Negligible", "pkgE"),
					newMatch("CVE-6", "Unknown", "pkgF"),
				},
			},
		},
	}

	report := summaryToReport(summary, manifest)

	assert.Equal(t, "kubescape-kubescape-deployment-storage-apiserver", report.Name)
	assert.Equal(t, "kubescape", report.Namespace)
	assert.Equal(t, managedByValue, report.Labels[managedByLabel])
	assert.Equal(t, sourceValue, report.Labels[sourceLabel])
	assert.Equal(t, 6, len(report.Results))

	// Scope points at the workload.
	expectedScope := &corev1.ObjectReference{Kind: "Deployment", Name: "storage", Namespace: "kubescape", APIVersion: "apps/v1"}
	if diff := cmp.Diff(expectedScope, report.Scope); diff != "" {
		t.Fatalf("unexpected scope (-want +got):\n%s", diff)
	}

	// critical+high -> fail(2), medium -> warn(1), low+negligible -> skip(2), unknown -> error(1).
	expectedSummary := policyreport.PolicyReportSummary{Fail: 2, Warn: 1, Skip: 2, Error: 1}
	if diff := cmp.Diff(expectedSummary, report.Summary); diff != "" {
		t.Fatalf("unexpected summary (-want +got):\n%s", diff)
	}
}

func newMatch(id, severity, pkg string) kubescape.Match {
	return kubescape.Match{
		Vulnerability: kubescape.Vulnerability{
			VulnerabilityMetadata: kubescape.VulnerabilityMetadata{ID: id, Severity: severity},
		},
		Artifact: kubescape.GrypePackage{Name: pkg, Version: "1.0"},
	}
}
