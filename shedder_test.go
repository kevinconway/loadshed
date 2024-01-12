// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"errors"
	"testing"
)

var shedErr error

func TestShedderDoNoRejection(t *testing.T) {
	t.Parallel()

	shed := NewShedder(
		OptionShedderRejectionRate(&staticCountingRate{value: 0.0}),
	)
	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}

	err := shed.Do(ctx, fn)
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func TestShedderDoNoRejectionUsingRule(t *testing.T) {
	t.Parallel()

	shed := NewShedder(
		OptionShedderRule(&staticRule{reject: false}),
	)
	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}

	err := shed.Do(ctx, fn)
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func TestShedderDoAppliesClassification(t *testing.T) {
	t.Parallel()

	cls := Classification("TEST")
	shed := NewShedder(
		OptionShedderRejectionRate(&staticCountingRate{value: 0.0}),
		OptionShedderClassifier(ClassifierFN(func(ctx context.Context) Classification { return cls })),
	)
	ctx := context.Background()
	found := false
	fn := func(ctx context.Context) error {
		found = ClassificationFromContext(ctx) == cls
		return nil
	}

	err := shed.Do(ctx, fn)
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
	if !found {
		t.Fatalf("the classification was not applied")
	}
}
func TestShedderDoAllRejection(t *testing.T) {
	t.Parallel()

	cls := Classification("TEST")
	shed := NewShedder(
		OptionShedderRejectionRate(&staticCountingRate{value: 1.0}),
		OptionShedderClassifier(ClassifierFN(func(ctx context.Context) Classification { return cls })),
	)

	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}
	err := shed.Do(ctx, fn)
	if err == nil {
		t.Fatal("expected ErrRejection but got nil")
	}
	var e ErrRejection
	if !errors.As(err, &e) {
		t.Fatalf("expected ErrRejection but got %s", err)
	}
	if e.Rule != RuleProbabilistic {
		t.Fatalf("expected rule %s but got %s", RuleProbabilistic, e.Rule)
	}
	if e.Classification != cls {
		t.Fatalf("expected classification %s but got %s", cls, e.Classification)
	}
	if e.Name != nameStatic {
		t.Fatalf("expected name %s but got %s", nameStatic, e.Name)
	}
	if e.Usage != 1.0 {
		t.Fatalf("expected %f usage but got %f", 1.0, e.Usage)
	}
	if e.Likelihood != 1.0 {
		t.Fatalf("expected %f likelihood but got %f", 1.0, e.Likelihood)
	}
	if e.Rate != 1.0 {
		t.Fatalf("expected %f rate but got %f", 1.0, e.Rate)
	}
}

func TestShedderDoAllRejectionUsingRule(t *testing.T) {
	t.Parallel()

	cls := Classification("TEST")
	shed := NewShedder(
		OptionShedderRule(&staticRule{reject: true}),
		OptionShedderClassifier(ClassifierFN(func(ctx context.Context) Classification { return cls })),
	)

	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}
	err := shed.Do(ctx, fn)
	if err == nil {
		t.Fatal("expected ErrRejection but got nil")
	}
	var e ErrRejection
	if !errors.As(err, &e) {
		t.Fatalf("expected ErrRejection but got %s", err)
	}
	if e.Rule != nameStatic {
		t.Fatalf("expected rule %s but got %s", nameStatic, e.Rule)
	}
	if e.Classification != cls {
		t.Fatalf("expected classification %s but got %s", cls, e.Classification)
	}
}

func BenchmarkShedderDoRate(b *testing.B) {
	shed := NewShedder(
		OptionShedderRejectionRate(&staticCountingRate{value: .5}),
	)
	ctx := context.Background()
	e := errors.New("TEST")
	fn := func(ctx context.Context) error {
		return e
	}
	for n := 0; n < b.N; n = n + 1 {
		shedErr = shed.Do(ctx, fn)
	}
}

func BenchmarkShedderWrapPlusSelect(b *testing.B) {
	shed := NewShedder(
		OptionShedderRejectionRate(&staticCountingRate{value: .5}),
	)
	ctx := context.Background()
	e := errors.New("TEST")
	fn := func(ctx context.Context) error {
		return e
	}
	fn = shed.WrapSelect(fn)
	for n := 0; n < b.N; n = n + 1 {
		shedErr = fn(ctx)
	}
}

const nameStatic = "static"

type staticCountingRate struct {
	value float32
	count int64
}

func (*staticCountingRate) Name(context.Context) string {
	return nameStatic
}

func (self *staticCountingRate) Usage(context.Context) float32 {
	return self.value
}

func (self *staticCountingRate) Likelihood(context.Context) float32 {
	return self.value
}

func (self *staticCountingRate) Rate(context.Context) float32 {
	return self.value
}

func (self *staticCountingRate) Wrap(fn Fn) Fn {
	return func(ctx context.Context) error {
		self.count = self.count + 1
		return fn(ctx)
	}
}

type staticRule struct {
	reject bool
}

func (self *staticRule) Name(ctx context.Context) string {
	return nameStatic
}
func (self *staticRule) Reject(ctx context.Context) bool {
	return self.reject
}
