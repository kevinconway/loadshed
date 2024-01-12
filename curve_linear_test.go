// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"testing"
)

func TestCurveLinear_Curve(t *testing.T) {
	t.Parallel()

	type fields struct {
		Lower    float32
		Upper    float32
		Exponent float32
	}
	tests := []struct {
		name   string
		fields fields
		value  float32
		want   float32
	}{
		{
			name: "0",
			fields: fields{
				Lower:    100,
				Upper:    200,
				Exponent: 1,
			},
			value: 50,
			want:  0,
		},
		{
			name: "1",
			fields: fields{
				Lower:    0,
				Upper:    20,
				Exponent: 1,
			},
			value: 50,
			want:  1,
		},
		{
			name: ".5",
			fields: fields{
				Lower:    0,
				Upper:    100,
				Exponent: 1,
			},
			value: 50,
			want:  .5,
		},
		{
			name: ".2",
			fields: fields{
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
			c := &CurveLinear{
				Upper:    tt.fields.Upper,
				Lower:    tt.fields.Lower,
				Exponent: tt.fields.Exponent,
			}
			result := c.Curve(ctx, tt.value)
			if result != tt.want {
				t.Errorf("CurveLinear.Curve() = %v, want %v", result, tt.want)
			}
		})
	}
}

var benchmarkCurveLinear float32

func BenchmarkCurveLinear_Curve(b *testing.B) {
	type fields struct {
		Lower    float32
		Upper    float32
		Exponent float32
	}
	tests := []struct {
		name   string
		fields fields
		value  float32
		want   float32
	}{
		{
			name: "0",
			fields: fields{
				Lower:    100,
				Upper:    200,
				Exponent: 1,
			},
			value: 50,
			want:  0,
		},
		{
			name: "1",
			fields: fields{
				Lower:    0,
				Upper:    20,
				Exponent: 1,
			},
			value: 50,
			want:  1,
		},
		{
			name: ".5",
			fields: fields{
				Lower:    0,
				Upper:    100,
				Exponent: 1,
			},
			value: 50,
			want:  .5,
		},
		{
			name: ".2",
			fields: fields{
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
			c := &CurveLinear{
				Upper:    tt.fields.Upper,
				Lower:    tt.fields.Lower,
				Exponent: tt.fields.Exponent,
			}
			b.ResetTimer()
			for n := 0; n < b.N; n = n + 1 {
				benchmarkCurveLinear = c.Curve(ctx, tt.value)
			}
		})
	}
}
