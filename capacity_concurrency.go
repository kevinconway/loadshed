// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"sync"
	"sync/atomic"
)

type OptionConcurrency func(*CapacityConcurrency)

func OptionConcurrencyName(name string) OptionConcurrency {
	return func(cc *CapacityConcurrency) {
		cc.name = name
	}
}

// CapacityConcurrency tracks the number of concurrent calls to a method.
type CapacityConcurrency struct {
	name    string
	limit   int32
	current *atomic.Int32
}

func NewCapacityConcurrency(limit int32, options ...OptionConcurrency) *CapacityConcurrency {
	c := &CapacityConcurrency{
		name:    defaultNameConcurrency,
		limit:   limit,
		current: &atomic.Int32{},
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}

func (self *CapacityConcurrency) Name(ctx context.Context) string {
	return self.name
}

func (self *CapacityConcurrency) Add(count int32) {
	self.current.Add(count)
}
func (self *CapacityConcurrency) Done(count int32) {
	self.current.Add(-count)
}

// Aggregate returns the current concurrency value.
func (self *CapacityConcurrency) Usage(ctx context.Context) float32 {
	return float32(float64(self.current.Load()) / float64(self.limit))
}

// Wrap a function in concurrency tracking.
func (self *CapacityConcurrency) Wrap(fn Fn) Fn {
	return func(ctx context.Context) error {
		self.Add(1)
		defer self.Done(1)
		var e = fn(ctx)
		return e
	}
}

// CapacityWaitGroup combines a CapacityConcurrency and a sync.WaitGroup. This
// type may be used in place of a wait group and can satisfy an interface
// matching the wait group's methods. Each delta given to Add() increases the
// reported concurrency count and each call to Done() decreases the count.
type CapacityWaitGroup struct {
	*CapacityConcurrency
	wg *sync.WaitGroup
}

func NewCapacityWaitGroup(limit int32, options ...OptionConcurrency) *CapacityWaitGroup {
	wrapped := NewCapacityConcurrency(limit, options...)
	return &CapacityWaitGroup{
		wg:                  &sync.WaitGroup{},
		CapacityConcurrency: wrapped,
	}
}

func (self *CapacityWaitGroup) Add(delta int) {
	self.wg.Add(delta)
	self.CapacityConcurrency.Add(int32(delta))
}

func (self *CapacityWaitGroup) Done() {
	self.wg.Done()
	self.CapacityConcurrency.Done(1)
}

func (self *CapacityWaitGroup) Wait() {
	self.wg.Wait()
}

const defaultNameConcurrency string = "CONCURRENCY"

var _ Capacity = &CapacityConcurrency{}
