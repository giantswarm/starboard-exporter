package ciskubebenchreport

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/giantswarm/starboard-exporter/utils"
)

const (
	metricNamespace = "starboard_exporter"
	metricSubsystem = "ciskubebenchreport"

	LabelGroupAll            = "all"
	labelGroupSummary        = "summary"
	labelGroupSectionSummary = "section"
	labelGroupResult         = "result"
)

type FieldScope string

const (
	FieldScopeReport  FieldScope = "report"
	FieldScopeSection FieldScope = "section"
	FieldScopeResult  FieldScope = "result"
)

type ReportLabel struct {
	Name   string
	Groups []string
	Scope  FieldScope
	// Handler valueFromReport
}

var metricLabels = []ReportLabel{
	{
		Name:   "node_name",
		Groups: []string{LabelGroupAll, labelGroupSummary, labelGroupSectionSummary, labelGroupResult},
		Scope:  FieldScopeReport,
	},
	{
		Name:   "pass_count",
		Groups: []string{LabelGroupAll, labelGroupSummary},
		Scope:  FieldScopeReport,
	},
	{
		Name:   "info_count",
		Groups: []string{LabelGroupAll, labelGroupSummary},
		Scope:  FieldScopeReport,
	},
	{
		Name:   "fail_count",
		Groups: []string{LabelGroupAll, labelGroupSummary},
		Scope:  FieldScopeReport,
	},
	{
		Name:   "warn_count",
		Groups: []string{LabelGroupAll, labelGroupSummary},
		Scope:  FieldScopeReport,
	},
	{
		Name:   "total_pass",
		Groups: []string{LabelGroupAll, labelGroupSectionSummary},
		Scope:  FieldScopeSection,
	},
	{
		Name:   "total_warn",
		Groups: []string{LabelGroupAll, labelGroupSectionSummary},
		Scope:  FieldScopeSection,
	},
	{
		Name:   "total_info",
		Groups: []string{LabelGroupAll, labelGroupSectionSummary},
		Scope:  FieldScopeSection,
	},
	{
		Name:   "total_fail",
		Groups: []string{LabelGroupAll, labelGroupSectionSummary},
		Scope:  FieldScopeSection,
	},
	{
		Name:   "node_type",
		Groups: []string{LabelGroupAll, labelGroupSectionSummary, labelGroupResult},
		Scope:  FieldScopeSection,
	},
	{
		Name:   "test_number",
		Groups: []string{LabelGroupAll, labelGroupResult},
		Scope:  FieldScopeResult,
	},
	{
		Name:   "test_desc",
		Groups: []string{LabelGroupAll, labelGroupResult},
		Scope:  FieldScopeResult,
	},
	{
		Name:   "test_status",
		Groups: []string{LabelGroupAll, labelGroupResult},
		Scope:  FieldScopeResult,
	},
}

func LabelWithName(name string) (label ReportLabel, ok bool) {
	for _, label := range metricLabels {
		if label.Name == name {
			return label, true
		}
	}
	return ReportLabel{}, false
}

func LabelsForGroup(group string) []ReportLabel {
	l := []ReportLabel{}
	for _, label := range metricLabels {
		if utils.SliceContains(label.Groups, group) {
			l = append(l, label)
		}
	}
	return l
}

func labelNamesForGroup(group string) []string {
	l := []string{}
	for _, label := range metricLabels {
		if utils.SliceContains(label.Groups, group) {
			l = append(l, label.Name)
		}
	}
	return l
}

func LabelNamesForList(list []ReportLabel) []string {
	l := []string{}
	for _, label := range list {
		l = append(l, label.Name)
	}
	return l
}

// Gauge for the count of all config audit rules summary
var (
	BenchmarkSummary = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "cis_benchmark_report_summary_count",
			Help:      "Exposes the summary of checks of a particular node.",
		},
		labelNamesForGroup(labelGroupSummary),
	)
)

var (
	BenchmarkSectionSummary = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "cis_benchmark_section_summary_count",
			Help:      "Exposes the summary of checks of a particular section on a particular node.",
		},
		labelNamesForGroup(labelGroupSectionSummary),
	)
)

var (
	BenchmarkTestInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "cis_benchmark_test_info",
			Help:      "Exposes the information of test of a particular section on a particular node.",
		},
		labelNamesForGroup(labelGroupResult),
	)
)

var CISBenchmarkInfo *prometheus.GaugeVec

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(BenchmarkSummary)
}
