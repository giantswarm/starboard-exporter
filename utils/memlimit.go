package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/go-logr/logr"
)

func SetupMemLimit(ctx context.Context, ratio float64, logger logr.Logger) error {
	if ratio <= 0 || ratio > 1.0 {
		return fmt.Errorf("value %f is invalid: ratio must be greater than 0 and less than or equal to 1", ratio)
	}

	limit, err := memlimit.Set(
		memlimit.WithRatio(ratio),
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
		memlimit.WithRefreshInterval(ctx, 5*time.Minute),
	)
	logger.Info("configured memlimit", "limit", limit)

	return err
}
