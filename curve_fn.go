// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
)

// CurveFN is an adapter for simple curving functions. For example:
//
//	CurveFN(func(ctx context.Context, value float32) float32 { return 0.0 })
type CurveFN func(ctx context.Context, value float32) float32

func (self CurveFN) Curve(ctx context.Context, value float32) float32 {
	return self(ctx, value)
}
