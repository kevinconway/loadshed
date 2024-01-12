// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"testing"
	"time"
)

func TestCapacityThrottle_Usage(t *testing.T) {
	t.Parallel()

	type fields struct {
		Duration time.Duration
	}
	tests := []struct {
		name          string
		fields        fields
		howLong       time.Duration
		expectedCalls int
	}{
		{
			name: "one hit",
			fields: fields{
				Duration: 2 * time.Millisecond,
			},
			howLong:       time.Millisecond,
			expectedCalls: 1,
		},
		{
			name: "two hits",
			fields: fields{
				Duration: 2 * time.Millisecond,
			},
			howLong:       3 * time.Millisecond,
			expectedCalls: 2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			wrapped := &capCounter{}
			a := NewCapacityThrottle(wrapped, tt.fields.Duration)
			start := time.Now()
			for time.Since(start) < tt.howLong {
				a.Usage(ctx)
				time.Sleep(tt.fields.Duration / 10)
			}
			if wrapped.calls != tt.expectedCalls {
				t.Fatalf("expected %d calls but got %d", tt.expectedCalls, wrapped.calls)
			}
		})
	}
}

var benchThrottleUsage float32

func BenchmarkCapacityThrottle(b *testing.B) {
	ctx := context.Background()
	wrapped := &capCounter{}
	self := NewCapacityThrottle(wrapped, time.Nanosecond) // low value to force cache refresh
	for n := 0; n < b.N; n = n + 1 {
		benchThrottleUsage = self.Usage(ctx)
	}
}

type capCounter struct {
	calls int
}

func (*capCounter) Name(context.Context) string {
	return "counter"
}

func (self *capCounter) Usage(context.Context) float32 {
	self.calls = self.calls + 1
	return 1
}
