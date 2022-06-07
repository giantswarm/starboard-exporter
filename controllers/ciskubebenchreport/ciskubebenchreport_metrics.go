package ciskubebenchreport

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/giantswarm/starboard-exporter/utils"
)

const (
	metricNamespace = "starboard_exporter"
	metricSubsystem = "ciskubebenchreport"

	LabelGroupAll     = "all"
	labelGroupSummary = "summary"
)

type FieldScope string

const (
	FieldScopeReport  FieldScope = "report"
	FieldScopeSection FieldScope = "section"
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
		Groups: []string{LabelGroupAll, labelGroupSummary},
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
			Name:      "cis_benchmark_summary_count",
			Help:      "Exposes the summary of checks of a particular node.",
		},
		labelNamesForGroup(labelGroupSummary),
	)
)

var CISBenchmarkInfo *prometheus.GaugeVec

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(BenchmarkSummary)
}
