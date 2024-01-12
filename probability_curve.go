// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
)

type FailureProbabilityCurve struct {
	Capacity
	Curve Curve
}

// NewFailureProbabilityCurveLinear generates a curving probability that uses
// the linear interpolation curve from CurveLinear.
func NewFailureProbabilityCurveLinear(cap Capacity, lower float32, upper float32, exponent float32) *FailureProbabilityCurve {
	return &FailureProbabilityCurve{
		Capacity: cap,
		Curve: &CurveLinear{
			Upper:    upper,
			Lower:    lower,
			Exponent: exponent,
		},
	}
}

// NewFailureProbabilityCurveIdentity generates a probability that returns the
// underlying capacity value without modifying it.
func NewFailureProbabilityCurveIdentity(cap Capacity) *FailureProbabilityCurve {
	return &FailureProbabilityCurve{
		Capacity: cap,
		Curve:    CurveFN(func(ctx context.Context, value float32) float32 { return value }),
	}
}

func (self *FailureProbabilityCurve) Likelihood(ctx context.Context) float32 {
	current := self.Capacity.Usage(ctx)
	return self.Curve.Curve(ctx, current)
}

func (self *FailureProbabilityCurve) Wrap(fn Fn) Fn {
	if w, ok := self.Capacity.(Wrapper); ok {
		return w.Wrap(fn)
	}
	return fn
}

var _ FailureProbability = &FailureProbabilityCurve{}
