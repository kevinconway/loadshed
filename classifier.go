// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import "context"

type classCtxKeyType struct{}

var classCtxKey = classCtxKeyType{} //nolint: gochecknoglobals

func ClassificationFromContext(ctx context.Context) Classification {
	v := ctx.Value(classCtxKey)
	if v == nil {
		return Classification("")
	}
	return v.(Classification)
}

func ClassificationToContext(ctx context.Context, class Classification) context.Context {
	return context.WithValue(ctx, classCtxKey, class)
}

// ClassifierFN is an adapter for simple classification functions. For example:
//
//	ClassifierFN(func(ctx context.Context) Classification {
//		requestPath := PathFromContext(ctx)
//		if requestPath == "/important/api" {
//			return Classification("HIGH")
//		}
//		return Classification("Normal")
//	})
type ClassifierFN func(ctx context.Context) Classification

func (self ClassifierFN) Classify(ctx context.Context) Classification {
	return self(ctx)
}
