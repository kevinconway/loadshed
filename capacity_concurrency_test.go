// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"testing"
	"time"
)

func TestAggregatorConcurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	done := make(chan interface{})
	fn := func(context.Context) error {
		<-done
		return nil
	}
	c := NewCapacityConcurrency(4)
	w := c.Wrap(fn)

	if c.Usage(ctx) != 0.0 {
		t.Fatalf("expected %d but got %f", 0, c.Usage(ctx))
	}
	go w(ctx)
	go w(ctx)
	time.Sleep(time.Millisecond) // force a context switch to allow goroutines to run.
	if c.Usage(ctx) != .5 {
		t.Fatalf("expected %f but got %f", .5, c.Usage(ctx))
	}
	go w(ctx)
	go w(ctx)
	time.Sleep(time.Millisecond)
	if c.Usage(ctx) != 1 {
		t.Fatalf("expected %f but got %f", 1.0, c.Usage(ctx))
	}
	close(done)
	time.Sleep(time.Millisecond)
	if c.Usage(ctx) != 0.0 {
		t.Fatalf("expected %f but got %f", 0.0, c.Usage(ctx))
	}
}

func TestAggregatorWaitGroup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := NewCapacityWaitGroup(4)
	done := make(chan interface{})
	fn := func(context.Context) error {
		defer c.Done()
		<-done
		return nil
	}

	if c.Usage(ctx) != 0 {
		t.Fatalf("expected %f but got %f", 0.0, c.Usage(ctx))
	}
	c.Add(2)
	go fn(ctx)
	go fn(ctx)
	time.Sleep(time.Millisecond) // force a context switch to allow goroutines to run.
	if c.Usage(ctx) != .5 {
		t.Fatalf("expected %f but got %f", .5, c.Usage(ctx))
	}
	c.Add(2)
	go fn(ctx)
	go fn(ctx)
	time.Sleep(time.Millisecond)
	if c.Usage(ctx) != 1 {
		t.Fatalf("expected %f but got %f", 1.0, c.Usage(ctx))
	}
	close(done)
	c.Wait()
	if c.Usage(ctx) != 0 {
		t.Fatalf("expected %f but got %f", 0.0, c.Usage(ctx))
	}
}

var benchConcurrencyErr error
var benchConcurrencyUsage float32

func BenchmarkCapacityConcurrency(b *testing.B) {
	ctx := context.Background()
	c := NewCapacityConcurrency(16)
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
