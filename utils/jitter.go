package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func Jitter(t time.Duration, maxJitterPercent int) (time.Duration, error) {
	// Get the maximum jitter length as a duration.
	maxJitter, err := time.ParseDuration(
		fmt.Sprintf("%dms",
			t.Milliseconds()*(int64(maxJitterPercent)/100)))
	if err != nil {
		return t, err
	}

	// Calcluate the minimum time we have to wait.
	minDuration := t - maxJitter

	// Set the final duration to the min + a random duration between 0 and our max jitter.
	return minDuration + time.Duration(rand.Int63n(int64(maxJitter))), nil // nolint:gosec // rand not used for crypto.
}
