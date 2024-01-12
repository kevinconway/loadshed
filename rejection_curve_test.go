// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"testing"
)

func TestRejectionCurve_Rate(t *testing.T) {
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
			wrapped := &staticProb{
				value: tt.value,
			}
			p := &RejectionRateCurve{
				FailureProbability: wrapped,
				defaultCurve:       tt.curve,
			}
			result := p.Rate(ctx)
			if result != tt.want {
				t.Errorf("RejectionRateCurve.Rate() = %v, want %v", result, tt.want)
			}
		})
	}
}

var benchmarkRateCurve float32

func BenchmarkRateCurve_Rate(b *testing.B) {
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
			wrapped := &staticProb{value: tt.want}
			p := &RejectionRateCurve{
				FailureProbability: wrapped,
				defaultCurve:       tt.curve,
			}
			b.ResetTimer()
			for n := 0; n < b.N; n = n + 1 {
				benchmarkRateCurve = p.Rate(ctx)
			}
		})
	}
}

type staticProb struct {
	value float32
}

func (*staticProb) Name(context.Context) string {
	return "static"
}

func (self *staticProb) Usage(context.Context) float32 {
	return self.value
}

func (self *staticProb) Likelihood(context.Context) float32 {
	return self.value
}

func (*staticProb) Wrap(fn Fn) Fn {
	return fn
}
