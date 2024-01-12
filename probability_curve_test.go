// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"testing"
)

func TestProbabilityCurve_Likelihood(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		curve Curve
		value float32
		want  float32
	}{
		{
			name: "0",
			curve: &CurveLinear{
				Lower:    100,
				Upper:    200,
				Exponent: 1,
			},
			value: 50,
			want:  0,
		},
		{
			name: "1",
			curve: &CurveLinear{
				Lower:    0,
				Upper:    20,
				Exponent: 1,
			},
			value: 50,
			want:  1,
		},
		{
			name: ".5",
			curve: &CurveLinear{
				Lower:    0,
				Upper:    100,
				Exponent: 1,
			},
			value: 50,
			want:  .5,
		},
		{
			name: ".2",
			curve: &CurveLinear{
				Lower:    80,
				Upper:    180,
				Exponent: 1,
			},
			value: 100,
			want:  .2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			wrapped := &staticCap{
				value: tt.value,
			}
			p := &FailureProbabilityCurve{
				Capacity: wrapped,
				Curve:    tt.curve,
			}
			result := p.Likelihood(ctx)
			if result != tt.want {
				t.Errorf("ProbabilityCurve.Likelihood() = %v, want %v", result, tt.want)
			}
		})
	}
}

var benchmarkProbabilityCurveLikelihood float32

func BenchmarkProbabilityCurve_Likelihood(b *testing.B) {
	tests := []struct {
		name  string
		curve Curve
		value float32
		want  float32
	}{
		{
			name: "0",
			curve: &CurveLinear{
				Lower:    100,
				Upper:    200,
				Exponent: 1,
			},
			value: 50,
			want:  0,
		},
		{
			name: "1",
			curve: &CurveLinear{
				Lower:    0,
				Upper:    20,
				Exponent: 1,
			},
			value: 50,
			want:  1,
		},
		{
			name: ".5",
			curve: &CurveLinear{
				Lower:    0,
				Upper:    100,
				Exponent: 1,
			},
			value: 50,
			want:  .5,
		},
		{
			name: ".2",
			curve: &CurveLinear{
				Lower:    80,
				Upper:    180,
				Exponent: 1,
			},
			value: 100,
			want:  .2,
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			ctx := context.Background()
			wrapped := &staticCap{value: tt.want}
			p := &FailureProbabilityCurve{
				Capacity: wrapped,
				Curve:    tt.curve,
			}
			b.ResetTimer()
			for n := 0; n < b.N; n = n + 1 {
				benchmarkProbabilityCurveLikelihood = p.Likelihood(ctx)
			}
		})
	}
}

type staticCap struct {
	value float32
}

func (a *staticCap) Name(context.Context) string {
	return "static"
}

func (a *staticCap) Usage(context.Context) float32 {
	return a.value
}

func (a *staticCap) Wrap(fn Fn) Fn {
	return fn
}
