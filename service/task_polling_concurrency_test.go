package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTaskPollingConcurrencyConstants(t *testing.T) {
	if taskPollingGlobalConcurrency != 30 {
		t.Fatalf("global task polling concurrency = %d, want 30", taskPollingGlobalConcurrency)
	}
	if taskPollingPerChannelConcurrency != 3 {
		t.Fatalf("per-channel task polling concurrency = %d, want 3", taskPollingPerChannelConcurrency)
	}
}

func TestTaskPollingLimiterPerChannelLimit(t *testing.T) {
	limiter := newTaskPollingLimiter(30, 3)
	releases := make([]func(), 0, 3)
	for range 3 {
		release, err := limiter.acquire(context.Background(), 1)
		if err != nil {
			t.Fatalf("acquire initial channel slot: %v", err)
		}
		releases = append(releases, release)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := limiter.acquire(ctx, 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("fourth channel slot error = %v, want deadline exceeded", err)
	}

	releases[0]()
	release, err := limiter.acquire(context.Background(), 1)
	if err != nil {
		t.Fatalf("acquire channel slot after release: %v", err)
	}
	release()
	for _, release := range releases[1:] {
		release()
	}
}

func TestTaskPollingLimiterGlobalLimit(t *testing.T) {
	limiter := newTaskPollingLimiter(30, 3)
	releases := make([]func(), 0, 30)
	for channelID := 1; channelID <= 10; channelID++ {
		for range 3 {
			release, err := limiter.acquire(context.Background(), channelID)
			if err != nil {
				t.Fatalf("acquire initial global slot: %v", err)
			}
			releases = append(releases, release)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := limiter.acquire(ctx, 11); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("31st global slot error = %v, want deadline exceeded", err)
	}

	for _, release := range releases {
		release()
	}
}
