// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestCapacityErrorRate_Usage(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name string
		Fn   func(context.Context) error
	}
	tests := []struct {
		name   string
		fields fields
		want   float32
	}{
		{
			name: "0%",
			fields: fields{
				Fn: func(context.Context) error { return nil },
			},
			want: 0.0,
		},
		{
			name: "100%",
			fields: fields{
				Fn: func(context.Context) error { return errors.New("") },
			},
			want: 1.0,
		},
		{
			name: "50%",
			fields: fields{
				Fn: func() func(context.Context) error {
					var count = -1
					return func(context.Context) error {
						count = count + 1
						if count%2 == 0 {
							return nil
						}
						return errors.New("")
					}
				}(),
			},
			want: 0.5,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			c := NewCapacityErrorRate()
			w := c.Wrap(tt.fields.Fn)
			for x := 0; x < 100; x = x + 1 {
				_ = w(ctx)
			}
			if got := c.Usage(ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CapacityErrorRate.Usage() = %v, want %v", got, tt.want)
			}
		})
	}
}

var benchErrRateErr error
var benchErrRateUsage float32
var benchErr = errors.New("benchmark")

func BenchmarkCapacityErrorRate(b *testing.B) {
	fn := func(context.Context) error {
		return benchErr
	}
	ctx := context.Background()
	for x := 1; x < 1000000; x = x * 10 {
		b.Run(fmt.Sprintf("%d", x), func(b *testing.B) {
			c := NewCapacityErrorRate(
				OptionErrorRateWindowBuckets(x),
				OptionErrorRateBucketDuration(time.Microsecond),
			)
			w := c.Wrap(fn)
			b.ResetTimer()
			for n := 0; n < b.N; n = n + 1 {
				benchErrRateErr = w(ctx)
				benchErrRateUsage = c.Usage(ctx)
			}
		})
	}
}
