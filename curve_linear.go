// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"math"
)

// CurveLinear calculates shifts the input based on the following formula:
//
//	f(x) = x < LOWER ? 0 : x > UPPER ? 1 : ((x - LOWER) / (UPPER - LOWER) )^EXPONENT
//
// The result is a linear interpolation of the usage value between the upper and
// lower limits, optionally modified by some exponent.
type CurveLinear struct {
	Upper    float32
	Lower    float32
	Exponent float32
}

func (self *CurveLinear) Curve(ctx context.Context, value float32) float32 {
	if value < self.Lower {
		return 0
	}
	if value > self.Upper {
		return 1
	}
	line := ((value - self.Lower) / (self.Upper - self.Lower))
	if self.Exponent != 1 {
		return float32(math.Pow(float64(line), float64(self.Exponent)))
	}
	return line
}

var _ Curve = &CurveLinear{}
