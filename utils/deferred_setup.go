package utils

import (
	"context"
	"time"

	"github.com/go-logr/logr"
)

// DeferredSetup is a manager.Runnable which waits until an API resource is
// served by the cluster and then runs Setup once.
//
// It covers report CRDs which are installed after the exporter starts, for
// example when trivy-operator is installed later than starboard-exporter.
// Without it, the controller for that resource would never be started.
type DeferredSetup struct {
	// Name identifies the controller in log messages.
	Name string
	// Interval is the time between two discovery checks.
	Interval time.Duration
	// IsServed reports whether the API resource is served by the cluster.
	IsServed func() (bool, error)
	// Setup sets up the controller. It runs at most once.
	Setup func() error
	Log   logr.Logger
}

// Start polls IsServed until it reports the resource as served, then runs
// Setup and returns its error. It returns nil when ctx ends first.
// Discovery errors are logged and retried, as the API server or an
// aggregated API may be unavailable for a moment.
func (d *DeferredSetup) Start(ctx context.Context) error {
	ticker := time.NewTicker(d.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		served, err := d.IsServed()
		if err != nil {
			d.Log.Error(err, "unable to discover API resource, will retry", "controller", d.Name)
			continue
		}
		if !served {
			continue
		}

		d.Log.Info("API resource is now served, starting controller", "controller", d.Name)
		return d.Setup()
	}
}
