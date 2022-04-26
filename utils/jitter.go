package utils

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// JitterRequeue accepts a default requeue time, a maximum percentage to jitter, and a logger,
// and returns a Result containing a requeue time which has been randomized with jitter.
func JitterRequeue(defaultDuration time.Duration, maxJitterPercent int, log logr.Logger) reconcile.Result {

	after, err := Jitter(defaultDuration, maxJitterPercent)
	if err != nil {
		log.Error(err, "Failed to calculate jitter")
		after = defaultDuration
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: after,
	}
}

// Jitter accepts a Duration and maximum percentage to jitter (as an int),
// and returns a random Duration in the range t +/- maxJitterPercent.
func Jitter(t time.Duration, maxJitterPercent int) (time.Duration, error) {
	// Get the maximum jitter length as a duration.
	// Max = t * maxJitterPercent / 100.

	// Decimal representation of the maximum jitter. E.g. 25% --> 0.25.
	jitterMultiplier := float64(maxJitterPercent) / 100.00

	// Maximum length of time, in milliseconds, which we can add or subtract from our target time.
	jitterDuration := int64(
		float64(t.Milliseconds()) * jitterMultiplier)

	// Maximum length of jitter time as a Go Duration.
	maxJitter, err := time.ParseDuration(fmt.Sprintf("%dms", jitterDuration))
	if err != nil {
		return t, err
	}

	// Calcluate the minimum time we have to wait.
	minDuration := t - maxJitter

	// Set the final duration to the min + a random duration between 0 and our max jitter.
	return minDuration + time.Duration(rand.Int63n(int64(maxJitter))), nil // nolint:gosec // rand not used for crypto.
}
