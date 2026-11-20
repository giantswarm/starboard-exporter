package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
)

func Test_DeferredSetup_setsUpOnceServed(t *testing.T) {
	// The API is not served, then discovery fails once, then it is served.
	answers := []struct {
		served bool
		err    error
	}{
		{served: false},
		{err: errors.New("discovery unavailable")},
		{served: true},
	}

	checks := 0
	setups := 0

	d := &DeferredSetup{
		Name:     "test",
		Interval: time.Millisecond,
		Log:      testr.New(t),
		IsServed: func() (bool, error) {
			a := answers[checks]
			checks++
			return a.served, a.err
		},
		Setup: func() error {
			setups++
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start returned an error: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("Start only returned because the context ended")
	}
	if checks != len(answers) {
		t.Errorf("expected %d discovery checks, got %d", len(answers), checks)
	}
	if setups != 1 {
		t.Errorf("expected Setup to run once, got %d", setups)
	}
}

func Test_DeferredSetup_stopsWithoutSetupWhenContextEnds(t *testing.T) {
	setups := 0

	d := &DeferredSetup{
		Name:     "test",
		Interval: time.Millisecond,
		Log:      testr.New(t),
		IsServed: func() (bool, error) { return false, nil },
		Setup: func() error {
			setups++
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start returned an error: %v", err)
	}
	if setups != 0 {
		t.Errorf("expected Setup not to run, got %d", setups)
	}
}

func Test_DeferredSetup_returnsSetupError(t *testing.T) {
	setupErr := errors.New("boom")

	d := &DeferredSetup{
		Name:     "test",
		Interval: time.Millisecond,
		Log:      testr.New(t),
		IsServed: func() (bool, error) { return true, nil },
		Setup:    func() error { return setupErr },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := d.Start(ctx); !errors.Is(err, setupErr) {
		t.Fatalf("expected the Setup error, got %v", err)
	}
}
