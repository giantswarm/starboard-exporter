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
		Name:   "report_name",
		Groups: []string{labelGroupSummary, labelGroupSectionSummary, labelGroupResult},
		Scope:  FieldScopeReport,
	},
	{
		Name:   "node_name",
		Groups: []string{labelGroupSummary, labelGroupSectionSummary, labelGroupResult},
		Scope:  FieldScopeReport,
	},
	{
		Name:   "status",
		Groups: []string{labelGroupSummary, labelGroupSectionSummary},
		Scope:  FieldScopeReport,
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
			Name:      "report_summary_count",
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
			Name:      "section_summary_count",
			Help:      "Exposes the summary of checks of a particular section on a particular node.",
		},
		labelNamesForGroup(labelGroupSectionSummary),
	)
)

var BenchmarkResultInfo *prometheus.GaugeVec

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(BenchmarkSummary)
	metrics.Registry.MustRegister(BenchmarkSectionSummary)
}
