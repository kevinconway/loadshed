// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"sync"
	"time"
)

// CapacityThrottle is a wrapper for other Capacity implementations that limits
// the number of times the underlying usage is calculated within a period of
// time. This exists to help amortize the cost of expensive capacity
// calculations when it is safe or desireable to do so.
type CapacityThrottle struct {
	Capacity
	duration time.Duration
	lock     *sync.Mutex
	last     time.Time
	cache    float32
	now      func() time.Time
	since    func(time.Time) time.Duration
}

func NewCapacityThrottle(wrapped Capacity, duration time.Duration) *CapacityThrottle {
	return &CapacityThrottle{
		Capacity: wrapped,
		duration: duration,
		lock:     &sync.Mutex{},
		now:      time.Now,
		since:    time.Since,
	}
}

// Usage returns from an internal cache until a duration has expired at which
// point it calls the wrapped Capacity to get a new value.
func (self *CapacityThrottle) Usage(ctx context.Context) float32 {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.since(self.last) >= self.duration {
		self.cache = self.Capacity.Usage(ctx)
		self.last = time.Now()
	}
	return self.cache
}

func (self *CapacityThrottle) Wrap(fn Fn) Fn {
	if wrap, ok := self.Capacity.(Wrapper); ok {
		return wrap.Wrap(fn)
	}
	return fn
}

var _ Capacity = &CapacityThrottle{}
