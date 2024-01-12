// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"time"

	"github.com/kevinconway/rolling/v3"
)

type OptionLandingRate func(*CapacityLandingRate)

func OptionLandingRateWindowBuckets(count int) OptionLandingRate {
	return func(clr *CapacityLandingRate) {
		clr.buckets = count
	}
}

func OptionLandingRateBucketDuration(d time.Duration) OptionLandingRate {
	return func(clr *CapacityLandingRate) {
		clr.bucketDuration = d
	}
}

func OptionLandingRateBucketSizeHint(size int) OptionLandingRate {
	return func(clr *CapacityLandingRate) {
		clr.bucketSizeHint = size
	}
}

func OptionLandingrateName(name string) OptionLandingRate {
	return func(clr *CapacityLandingRate) {
		clr.name = name
	}
}

// CapacityLandingRate considers the number of method invocations within a window of
// time that have begun. Note that this counts all attempts to invoke a method
// and does not distinguish success or failure.
//
// The rate calculation is based on a rolling window. The default size of the
// window is 1s with each bucket representing 10ms. Both of these values can
// be modified using constructor options.
type CapacityLandingRate struct {
	name           string
	invocations    landingRateWindow
	limit          int
	buckets        int
	bucketDuration time.Duration
	bucketSizeHint int
}

func NewCapacityLandingRate(limit int, options ...OptionLandingRate) *CapacityLandingRate {
	c := &CapacityLandingRate{
		name:           defaultNameLandingRate,
		limit:          limit,
		buckets:        100,
		bucketDuration: 10 * time.Millisecond,
		bucketSizeHint: 0,
	}
	for _, opt := range options {
		opt(c)
	}
	w := rolling.NewPreallocatedWindow[int](c.buckets, c.bucketSizeHint)
	c.invocations = rolling.NewTimePolicyConcurrent[int](w, c.bucketDuration)
	return c
}

func (self *CapacityLandingRate) Name(context.Context) string {
	return self.name
}

func (self *CapacityLandingRate) Usage(ctx context.Context) float32 {
	total := self.invocations.Reduce(ctx, rolling.Count[int])
	return float32(float64(total) / float64(self.limit))
}

func (self *CapacityLandingRate) Wrap(fn Fn) Fn {
	return func(ctx context.Context) error {
		self.invocations.Append(ctx, 1)
		return fn(ctx)
	}
}

type landingRateWindow interface {
	Append(ctx context.Context, v int)
	Reduce(ctx context.Context, r rolling.Reduction[int]) int
}

const defaultNameLandingRate string = "LANDING RATE"

var _ Capacity = &CapacityLandingRate{}
