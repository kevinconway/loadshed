// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kevinconway/rolling/v3"
)

func TestAggregatorLatency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fn := func(context.Context) error {
		time.Sleep(time.Millisecond)
		return nil
	}
	c := NewCapacityLatency(2 * time.Millisecond)
	w := c.Wrap(fn)

	if c.Usage(ctx) != 0.0 {
		t.Fatalf("expected %f but got %f", 0.0, c.Usage(ctx))
	}
	for x := 0; x < 100; x = x + 1 {
		_ = w(ctx)
	}
	u := c.Usage(ctx)
	if u < .5 || u > .65 {
		// This test has to account for jitter in the actual execution time so
		// the range is fairly broad.
		t.Fatalf("expected %f but got %f", .5, c.Usage(ctx))
	}
}

var aggLatencyErr error
var aggLatencyAgg float32

func BenchmarkAggregatorLatencyAvg(b *testing.B) {
	fn := func(context.Context) error {
		return nil
	}
	ctx := context.Background()
	for x := 1; x < 1000000; x = x * 10 {
		b.Run(fmt.Sprintf("%d", x), func(b *testing.B) {
			c := NewCapacityLatency(time.Millisecond, OptionLatencyWindowBuckets(x))
			w := c.Wrap(fn)
			b.ResetTimer()
			for n := 0; n < b.N; n = n + 1 {
				aggLatencyErr = w(ctx)
				aggLatencyAgg = c.Usage(ctx)
			}
		})
	}
}

func BenchmarkAggregatorLatencyAvgMeasurePanics(b *testing.B) {
	fn := func(context.Context) error {
		return nil
	}
	ctx := context.Background()
	for x := 1; x < 1000000; x = x * 10 {
		b.Run(fmt.Sprintf("%d", x), func(b *testing.B) {
			c := NewCapacityLatency(time.Millisecond, OptionLatencyWindowBuckets(x), OptionLatencyMeasurePanics(true))
			w := c.Wrap(fn)
			b.ResetTimer()
			for n := 0; n < b.N; n = n + 1 {
				aggLatencyErr = w(ctx)
				aggLatencyAgg = c.Usage(ctx)
			}
		})
	}
}

func BenchmarkLatencyPercentile(b *testing.B) {
	fn := func(context.Context) error {
		return nil
	}
	ctx := context.Background()
	for x := 1; x < 1000000; x = x * 10 {
		b.Run(fmt.Sprintf("%d", x), func(b *testing.B) {
			c := NewCapacityLatency(time.Millisecond, OptionLatencyWindowBuckets(x), OptionLatencyReduction(rolling.FastPercentile[time.Duration](99.9)))
			w := c.Wrap(fn)
			b.ResetTimer()
			for n := 0; n < b.N; n = n + 1 {
				aggLatencyErr = w(ctx)
				aggLatencyAgg = c.Usage(ctx)
			}
		})
	}
}
