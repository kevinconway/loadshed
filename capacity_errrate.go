// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"math"
	"time"

	"github.com/kevinconway/rolling/v3"
)

type OptionErrorRate func(*CapacityErrorRate)

func OptionErrorRateWindowBuckets(count int) OptionErrorRate {
	return func(cer *CapacityErrorRate) {
		cer.buckets = count
	}
}

func OptionErrorRateBucketDuration(d time.Duration) OptionErrorRate {
	return func(cer *CapacityErrorRate) {
		cer.bucketDuration = d
	}
}

func OptionErrorRateBucketSizeHint(size int) OptionErrorRate {
	return func(cer *CapacityErrorRate) {
		cer.bucketSizeHint = size
	}
}

func OptionErrorRateMinimumPoints(min int) OptionErrorRate {
	return func(cer *CapacityErrorRate) {
		cer.minimumPoints = min
	}
}

func OptionErrorRateName(name string) OptionErrorRate {
	return func(cer *CapacityErrorRate) {
		cer.name = name
	}
}

// CapacityErrorRate calculates the percent error of invocations within a winow
// of time. Returned errors and panics are both considered in the rate
// calculation.
//
// Attempts and errors are always recorded within the same bucket of the window.
// The rate is then calculated as (errors / attempts) within the window. The
// current rate is given as the current capacity usage value.
//
// The rate calculation is based on a rolling window. The default size of the
// window is 1s with each bucket representing 10ms. Both of these values can
// be modified using constructor options.
type CapacityErrorRate struct {
	name           string
	invocations    errRateWindow
	errors         errRateWindow
	buckets        int
	bucketDuration time.Duration
	bucketSizeHint int
	minimumPoints  int
	attemptReducer rolling.Reduction[int]
	errReducer     rolling.Reduction[int]
}

func NewCapacityErrorRate(options ...OptionErrorRate) *CapacityErrorRate {
	c := &CapacityErrorRate{
		name:           defaultNameErrorRate,
		buckets:        100,
		bucketDuration: 10 * time.Millisecond,
		bucketSizeHint: 0,
		minimumPoints:  0,
	}
	for _, opt := range options {
		opt(c)
	}
	w := rolling.NewPreallocatedWindow[int](c.buckets, c.bucketSizeHint)
	c.invocations = rolling.NewTimePolicyConcurrent[int](w, c.bucketDuration)
	w = rolling.NewPreallocatedWindow[int](c.buckets, c.bucketSizeHint)
	c.errors = rolling.NewTimePolicyConcurrent[int](w, c.bucketDuration)
	c.attemptReducer = rolling.Count[int]
	if c.minimumPoints > 0 {
		c.attemptReducer = rolling.MinimumPoints[int](c.minimumPoints, rolling.Count[int])
	}
	c.errReducer = rolling.Count[int]
	return c
}

func (self *CapacityErrorRate) Name(context.Context) string {
	return self.name
}

func (self *CapacityErrorRate) Usage(ctx context.Context) float32 {
	attempts := self.invocations.Reduce(ctx, self.attemptReducer)
	if attempts == 0 {
		return 0.0
	}
	errors := self.errors.Reduce(ctx, self.errReducer)
	value := float64(errors) / float64(attempts)
	if math.IsNaN(value) {
		value = 0.0
	}
	return float32(value)
}

func (self *CapacityErrorRate) Wrap(fn Fn) Fn {
	return func(ctx context.Context) error {
		var e error
		didPanic := true
		defer func() {
			self.invocations.Append(ctx, 1)
			if e != nil || didPanic == true {
				self.errors.Append(ctx, 1)
			}
		}()
		e = fn(ctx)
		didPanic = false
		return e
	}
}

type errRateWindow interface {
	Append(ctx context.Context, v int)
	Reduce(ctx context.Context, r rolling.Reduction[int]) int
}

const defaultNameErrorRate string = "ERROR RATE"

var _ Capacity = &CapacityErrorRate{}
