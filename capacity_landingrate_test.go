// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"testing"
	"time"
)

func TestCapacityLandingRate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fn := func(context.Context) error {
		return nil
	}
	c := NewCapacityLandingRate(4, OptionLandingRateWindowBuckets(10), OptionLandingRateBucketDuration(time.Millisecond))
	w := c.Wrap(fn)

	if c.Usage(ctx) != 0.0 {
		t.Fatalf("expected %d but got %f", 0, c.Usage(ctx))
	}
	w(ctx)
	w(ctx)
	if c.Usage(ctx) != .5 {
		t.Fatalf("expected %f but got %f", .5, c.Usage(ctx))
	}
	w(ctx)
	w(ctx)
	if c.Usage(ctx) != 1 {
		t.Fatalf("expected %f but got %f", 1.0, c.Usage(ctx))
	}
	time.Sleep(10 * time.Millisecond)
	if c.Usage(ctx) != 0.0 {
		t.Fatalf("expected %f but got %f", 0.0, c.Usage(ctx))
	}
}

var benchLandingErr error
var benchLandingUsage float32

func BenchmarkCapacityLandingRate(b *testing.B) {
	ctx := context.Background()
	c := NewCapacityLandingRate(16)
	fn := func(context.Context) error {
		return nil
	}
	w := c.Wrap(fn)
	b.ResetTimer()
	for n := 0; n < b.N; n = n + 1 {
		benchConcurrencyErr = w(ctx)
		benchConcurrencyUsage = c.Usage(ctx)
	}
}
