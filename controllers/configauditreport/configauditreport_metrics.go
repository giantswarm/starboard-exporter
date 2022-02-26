package configauditreport

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	metricNamespace = "starboard_exporter"
	metricSubsystem = "configauditreport"
)

var metricLabels = []string{
	"resource_name",
	"resource_namespace",
	"severity",
}

// Gauge for the count of all config audit rules summary
var (
	ConfigAuditSummary = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "resource_checks_summary_count",
			Help:      "Exposes the number of checks of a particular severity.",
		},
		metricLabels,
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(ConfigAuditSummary)
}
