// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
)

type RejectionRateCurve struct {
	FailureProbability
	defaultCurve         Curve
	classificationCurves map[Classification]Curve
}

// NewRejectionRateCurveLinear generates a curving rejection rate that uses the
// linear interpolation curve from CurveLinear.
func NewRejectionRateCurveLinear(probability FailureProbability, lower float32, upper float32, exponent float32) *RejectionRateCurve {
	return &RejectionRateCurve{
		FailureProbability: probability,
		defaultCurve: &CurveLinear{
			Upper:    upper,
			Lower:    lower,
			Exponent: exponent,
		},
		classificationCurves: map[Classification]Curve{},
	}
}

// NewRejectionRateCurveIdentity generates a rejection rate that returns the
// underlying failure probability value without modifying it.
func NewRejectionRateCurveIdentity(probability FailureProbability) *RejectionRateCurve {
	return &RejectionRateCurve{
		FailureProbability:   probability,
		defaultCurve:         CurveFN(func(ctx context.Context, value float32) float32 { return value }),
		classificationCurves: map[Classification]Curve{},
	}
}

// NewRejectionRateCurveByClassification allows for the failure probability to
// be translated to a rejection rate based on the classification of an
// invocation. For example, LOW priority requests can have a higher rejection
// rate for the same failure probability compared to HIGH priority.
func NewRejectionRateCurveByClassification(probability FailureProbability, defaultCurve Curve, classes map[Classification]Curve) *RejectionRateCurve {
	return &RejectionRateCurve{
		FailureProbability:   probability,
		defaultCurve:         defaultCurve,
		classificationCurves: classes,
	}
}

func (self *RejectionRateCurve) Rate(ctx context.Context) float32 {
	curve := self.classificationCurves[ClassificationFromContext(ctx)]
	if curve == nil {
		curve = self.defaultCurve
	}
	current := self.FailureProbability.Likelihood(ctx)
	return curve.Curve(ctx, current)
}

func (self *RejectionRateCurve) Wrap(fn Fn) Fn {
	if w, ok := self.FailureProbability.(Wrapper); ok {
		return w.Wrap(fn)
	}
	return fn
}

var _ RejectionRate = &RejectionRateCurve{}
