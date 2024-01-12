// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"time"

	"github.com/kevinconway/rolling/v3"
)

type OptionLatency func(*CapacityLatency)

// OptionLatencyReduction sets the reduction method used when calculating usage
// value of the window of data. The default value is an average function.
func OptionLatencyReduction(r LatencyReduction) OptionLatency {
	return func(cl *CapacityLatency) {
		cl.reduction = r
	}
}

// OptionLatencyMeasurePanics modifies the capacity to capture latency for
// executions that resulted in a panic in addition to executions that exit
// normally. The default value is false.
func OptionLatencyMeasurePanics(v bool) OptionLatency {
	return func(cl *CapacityLatency) {
		cl.measurePanics = v
	}
}

func OptionLatencyWindowBuckets(count int) OptionLatency {
	return func(cl *CapacityLatency) {
		cl.buckets = count
	}
}

func OptionLatencyBucketDuration(d time.Duration) OptionLatency {
	return func(cl *CapacityLatency) {
		cl.bucketDuration = d
	}
}

func OptionLatencyBucketSizeHint(size int) OptionLatency {
	return func(cl *CapacityLatency) {
		cl.bucketSizeHint = size
	}
}

func OptionLatencyMinimumPoints(min int) OptionLatency {
	return func(cl *CapacityLatency) {
		cl.minimumPoints = min
	}
}

func OptionLatencyName(name string) OptionLatency {
	return func(cl *CapacityLatency) {
		cl.name = name
	}
}

// CapacityLatency calculates the execution time of method invocations within a
// winow of time.
//
// By default, latency is calculated by taking an average of method invocation
// time within the window. You can provide an alternative calculation using the
// OptionCapacityLatencyReduction option.
//
// The latency calculation is based on a rolling window. The default size of the
// window is 1s with each bucket representing 10ms. Both of these values can
// be modified using constructor options.
type CapacityLatency struct {
	name           string
	window         latencyWindow
	limit          time.Duration
	buckets        int
	bucketDuration time.Duration
	bucketSizeHint int
	minimumPoints  int
	reduction      LatencyReduction
	measurePanics  bool
}

// NewCapacityLatency defaults to installing an average function for the
// window reduction. Use the AggregatorLatencyOptionReduction modifier to set
// a different reduction.
func NewCapacityLatency(limit time.Duration, options ...OptionLatency) *CapacityLatency {
	c := &CapacityLatency{
		name:           defaultNameLatency,
		limit:          limit,
		buckets:        100,
		bucketDuration: 10 * time.Millisecond,
		bucketSizeHint: 0,
		minimumPoints:  0,
		reduction:      rolling.Avg[time.Duration],
		measurePanics:  false,
	}
	for _, option := range options {
		option(c)
	}
	w := rolling.NewPreallocatedWindow[time.Duration](c.buckets, c.bucketSizeHint)
	c.window = rolling.NewTimePolicyConcurrent[time.Duration](w, c.bucketDuration)
	return c
}

func (self *CapacityLatency) Name(context.Context) string {
	return self.name
}

// Append adds a latency measure to the underlying window.
func (self *CapacityLatency) Append(ctx context.Context, v time.Duration) {
	self.window.Append(ctx, v)
}

func (self *CapacityLatency) Usage(ctx context.Context) float32 {
	value := self.window.Reduce(ctx, self.reduction)
	return float32(value.Seconds() / self.limit.Seconds())
}

func (self *CapacityLatency) Wrap(fn Fn) Fn {
	return func(ctx context.Context) error {
		start := time.Now()
		if self.measurePanics {
			defer func() {
				d := time.Since(start)
				self.Append(ctx, d)
			}()
		}
		e := fn(ctx)
		if !self.measurePanics {
			d := time.Since(start)
			self.Append(ctx, d)
		}
		return e
	}
}

type LatencyReduction = rolling.Reduction[time.Duration]

type latencyWindow interface {
	Append(ctx context.Context, v time.Duration)
	Reduce(ctx context.Context, r rolling.Reduction[time.Duration]) time.Duration
}

const defaultNameLatency string = "LATENCY"

var _ Capacity = &CapacityLatency{}
